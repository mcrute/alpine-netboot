package cmd

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"code.crute.us/mcrute/golib/cli"
	"code.crute.us/mcrute/golib/clients/netbox/v4"
	"code.crute.us/mcrute/golib/secrets"
	"code.crute.us/mcrute/netboot-server/app"
	"code.crute.us/mcrute/netboot-server/netboxconfig"
	"code.crute.us/mcrute/netboot-server/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type App struct {
	TftpBoot     fs.FS
	IpxeTemplate string
}

func getNetboxKey(ctx context.Context, path string) (string, error) {
	vc, err := secrets.NewVaultClient(&secrets.VaultClientConfig{})
	if err != nil {
		return "", err
	}

	if err = vc.Authenticate(ctx); err != nil {
		return "", err
	}

	key := &secrets.ApiKey{}
	if _, err := vc.Secret(ctx, path, &key); err != nil {
		return "", err
	}

	return key.Key, nil
}

func (a *App) Main(c *cobra.Command, args []string) {
	//
	// Load Config
	//
	appCfg := app.Config{}
	cli.MustGetConfig(c, &appCfg)

	//
	// Setup Logger
	//
	lcfg := zap.NewProductionConfig()
	if appCfg.Debug {
		lcfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	logger, err := lcfg.Build()
	if err != nil {
		log.Fatalf("Error configuring zap logger: %s", err)
	}
	defer logger.Sync()

	//
	// Setup root context and signals
	//
	wg := &sync.WaitGroup{}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer cancel()

	//
	// Setup TFTP Server
	//
	tftpServer := &util.TftpServer{
		Addr:   appCfg.BindTftp,
		Logger: logger,
		ReadHandler: &app.TftpHandler{
			Root: util.MustSub(a.TftpBoot, "tftpboot"),
		},
	}

	//
	// Setup HTTP Server
	//
	mux := http.NewServeMux()
	httpServer := &util.HttpServer{
		Addr:    appCfg.BindHttp,
		Logger:  logger,
		Handler: mux,
	}

	//
	// Setup Distribution Catalog
	//
	catalogErrors := make(chan error, 10)
	catalog, err := app.LoadDistributionCatalog(os.DirFS(appCfg.DistroFilesPath), catalogErrors, logger)
	if err != nil {
		logger.Fatal("Error creating initial distro catalog", zap.Error(err))
	}
	catalog.ManageAsync(ctx, wg)

	//
	// Setup IPXE Render Handler
	//
	varsCfg, err := app.LoadVarsConfigYaml(appCfg.VarsConfigFile)
	if err != nil {
		logger.Fatal("Error loading variables configuration", zap.Error(err))
	}

	ipxeRendererHandler := &app.IpxeRendererHandler{
		Logger:       logger,
		VarsConfig:   varsCfg,
		NtpServer:    appCfg.NtpServer,
		HttpServer:   appCfg.HttpServer,
		CatalogWatch: make(chan app.DistroList, 1),
	}
	if err := ipxeRendererHandler.ParseTemplate(a.IpxeTemplate); err != nil {
		logger.Fatal("Error parsing IPXE template", zap.Error(err))
	}
	catalog.Watch(ipxeRendererHandler.CatalogWatch)
	ipxeRendererHandler.WatchCatalogAsync(ctx, wg)

	//
	// Setup AKOVL Handler
	//
	netboxKey, err := getNetboxKey(ctx, appCfg.VaultNetboxPath)
	if err != nil {
		logger.Fatal("Error getting Netbox key from Vault", zap.Error(err))
	}

	apkOvlHandler := &app.ApkOvlHandler{
		Logger: logger,
		Coordinator: &netboxconfig.ConfigCoordinator{
			DefaultConfigId: appCfg.NetboxDefaultConfigId,
			NetboxClient: &netbox.BasicNetboxClient{
				NetboxHttpClient: netbox.MustNewNetboxHttpClient(netboxKey, appCfg.NetboxHost),
			},
		},
	}

	//
	// Add HTTP Routes
	//
	mux.Handle("GET /boot.ipxe", &app.IpxeRedirectHandler{HttpServer: appCfg.HttpServer})
	mux.Handle("GET /{mac}/boot.ipxe", ipxeRendererHandler)
	mux.Handle("GET /{mac}/apkovl.tar.gz", apkOvlHandler)
	mux.Handle("GET /distros/*", catalog)
	mux.Handle("GET /tftpboot/*", http.FileServerFS(a.TftpBoot))
	mux.Handle("GET /", &app.WebIndexHandler{})

	//
	// Run Servers
	//
	tftpServer.ListenAndServeAsync()
	httpServer.ListenAndServeAsync()

	//
	// Await termination and clean up servers
	//
	terminateAndCleanup := func() {
		tftpServer.Shutdown(ctx)
		httpServer.Shutdown(ctx)
		wg.Wait()
	}

	select {
	case err := <-catalogErrors:
		logger.Error("Error scanning catalog, terminating", zap.Error(err))
		terminateAndCleanup()
	case <-ctx.Done():
		logger.Info("Shutdown requested, terminating")
		terminateAndCleanup()
	}
}

func CmdMain(tftpboot fs.FS, ipxeTemplate string) {
	cmd := &App{
		TftpBoot:     tftpboot,
		IpxeTemplate: ipxeTemplate,
	}

	rootCmd := &cobra.Command{
		Use:   "bootstrap-server",
		Short: "Netboot Bootstrap Server",
		Run:   cmd.Main,
	}
	cli.AddFlags(rootCmd, &app.Config{}, app.DefaultConfig, "")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error running root command: %s", err)
	}
}
