package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/telhawk-systems/telhawk-stack/core/internal/config"
	"github.com/telhawk-systems/telhawk-stack/core/internal/dlq"
	"github.com/telhawk-systems/telhawk-stack/core/internal/handlers"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer"
	"github.com/telhawk-systems/telhawk-stack/core/internal/normalizer/generated"
	"github.com/telhawk-systems/telhawk-stack/core/internal/pipeline"
	"github.com/telhawk-systems/telhawk-stack/core/internal/server"
	"github.com/telhawk-systems/telhawk-stack/core/internal/service"
	"github.com/telhawk-systems/telhawk-stack/core/internal/storage"
	"github.com/telhawk-systems/telhawk-stack/core/internal/validator"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file")
	addr := flag.String("addr", "", "override listen address")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	if *addr != "" {
		listenAddr = *addr
	}

	// Initialize all normalizers with OCSF passthrough first
	registry := normalizer.NewRegistry(
		// OCSF Passthrough (for events already in OCSF format)
		normalizer.OCSFPassthroughNormalizer{},
		
		// Application (8)
		generated.NewApiActivityNormalizer(),
		generated.NewApplicationErrorNormalizer(),
		generated.NewApplicationLifecycleNormalizer(),
		generated.NewDatastoreActivityNormalizer(),
		generated.NewFileHostingNormalizer(),
		generated.NewScanActivityNormalizer(),
		generated.NewWebResourceAccessActivityNormalizer(),
		generated.NewWebResourcesActivityNormalizer(),
		
		// Discovery (24)
		generated.NewAdminGroupQueryNormalizer(),
		generated.NewCloudResourcesInventoryInfoNormalizer(),
		generated.NewConfigStateNormalizer(),
		generated.NewDeviceConfigStateChangeNormalizer(),
		generated.NewDiscoveryResultNormalizer(),
		generated.NewEvidenceInfoNormalizer(),
		generated.NewFileQueryNormalizer(),
		generated.NewFolderQueryNormalizer(),
		generated.NewInventoryInfoNormalizer(),
		generated.NewJobQueryNormalizer(),
		generated.NewKernelObjectQueryNormalizer(),
		generated.NewModuleQueryNormalizer(),
		generated.NewNetworkConnectionQueryNormalizer(),
		generated.NewNetworksQueryNormalizer(),
		generated.NewOsintInventoryInfoNormalizer(),
		generated.NewPatchStateNormalizer(),
		generated.NewPeripheralDeviceQueryNormalizer(),
		generated.NewProcessQueryNormalizer(),
		generated.NewServiceQueryNormalizer(),
		generated.NewSessionQueryNormalizer(),
		generated.NewSoftwareInfoNormalizer(),
		generated.NewStartupItemQueryNormalizer(),
		generated.NewUserInventoryNormalizer(),
		generated.NewUserQueryNormalizer(),
		
		// Findings (9)
		generated.NewApplicationSecurityPostureFindingNormalizer(),
		generated.NewComplianceFindingNormalizer(),
		generated.NewDataSecurityFindingNormalizer(),
		generated.NewDetectionFindingNormalizer(),
		generated.NewFindingNormalizer(),
		generated.NewIamAnalysisFindingNormalizer(),
		generated.NewIncidentFindingNormalizer(),
		generated.NewSecurityFindingNormalizer(),
		generated.NewVulnerabilityFindingNormalizer(),
		
		// IAM (6)
		generated.NewAccountChangeNormalizer(),
		generated.NewAuthenticationNormalizer(),
		generated.NewAuthorizeSessionNormalizer(),
		generated.NewEntityManagementNormalizer(),
		generated.NewGroupManagementNormalizer(),
		generated.NewUserAccessNormalizer(),
		
		// Network (14)
		generated.NewDhcpActivityNormalizer(),
		generated.NewDnsActivityNormalizer(),
		generated.NewEmailActivityNormalizer(),
		generated.NewEmailFileActivityNormalizer(),
		generated.NewEmailUrlActivityNormalizer(),
		generated.NewFtpActivityNormalizer(),
		generated.NewHttpActivityNormalizer(),
		generated.NewNetworkActivityNormalizer(),
		generated.NewNetworkFileActivityNormalizer(),
		generated.NewNtpActivityNormalizer(),
		generated.NewRdpActivityNormalizer(),
		generated.NewSmbActivityNormalizer(),
		generated.NewSshActivityNormalizer(),
		generated.NewTunnelActivityNormalizer(),
		
		// Remediation (4)
		generated.NewFileRemediationActivityNormalizer(),
		generated.NewNetworkRemediationActivityNormalizer(),
		generated.NewProcessRemediationActivityNormalizer(),
		generated.NewRemediationActivityNormalizer(),
		
		// System (10)
		generated.NewEventLogActvityNormalizer(),
		generated.NewFileActivityNormalizer(),
		generated.NewKernelActivityNormalizer(),
		generated.NewKernelExtensionActivityNormalizer(),
		generated.NewMemoryActivityNormalizer(),
		generated.NewModuleActivityNormalizer(),
		generated.NewPeripheralActivityNormalizer(),
		generated.NewProcessActivityNormalizer(),
		generated.NewScheduledJobActivityNormalizer(),
		generated.NewScriptActivityNormalizer(),
		
		// Unmanned Systems (2)
		generated.NewAirborneBroadcastActivityNormalizer(),
		generated.NewDroneFlightsActivityNormalizer(),
		
		// Fallback for generic HEC events
		normalizer.HECNormalizer{},
	)
	log.Printf("Registered %d normalizers (77 generated + 1 fallback)", 77+1)
	validators := validator.NewChain(validator.BasicValidator{})
	pipe := pipeline.New(registry, validators)
	storageClient := storage.NewClient(cfg.Storage.URL)
	
	// Initialize DLQ if enabled
	var dlqQueue *dlq.Queue
	if cfg.DLQ.Enabled {
		var err error
		dlqQueue, err = dlq.NewQueue(cfg.DLQ.BasePath)
		if err != nil {
			log.Printf("WARN: failed to initialize DLQ: %v (continuing without DLQ)", err)
		} else {
			log.Printf("DLQ enabled at %s", cfg.DLQ.BasePath)
		}
	} else {
		log.Println("DLQ disabled")
	}
	
	processor := service.NewProcessor(pipe, storageClient, dlqQueue)
	handler := handlers.NewProcessorHandler(processor)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      server.NewRouter(handler),
		ReadTimeout:  cfg.Server.ReadTimeout(),
		WriteTimeout: cfg.Server.WriteTimeout(),
		IdleTimeout:  cfg.Server.IdleTimeout(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("core service listening on %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
