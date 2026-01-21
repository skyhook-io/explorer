package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/skyhook-io/skyhook-explorer/internal/helm"
	"github.com/skyhook-io/skyhook-explorer/internal/k8s"
	"github.com/skyhook-io/skyhook-explorer/internal/server"
	"github.com/skyhook-io/skyhook-explorer/internal/static"
)

var (
	version = "dev"
)

func main() {
	// Parse flags
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file (default: ~/.kube/config)")
	namespace := flag.String("namespace", "", "Initial namespace filter (empty = all namespaces)")
	port := flag.Int("port", 9280, "Server port")
	noBrowser := flag.Bool("no-browser", false, "Don't auto-open browser")
	devMode := flag.Bool("dev", false, "Development mode (serve frontend from filesystem)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	persistHistory := flag.Bool("persist-history", false, "Persist change history to file")
	historyLimit := flag.Int("history-limit", 1000, "Maximum number of changes to retain in history")
	flag.Parse()

	if *showVersion {
		fmt.Printf("skyhook-explorer %s\n", version)
		os.Exit(0)
	}

	log.Printf("Skyhook Explorer %s starting...", version)

	// Initialize K8s client
	err := k8s.Initialize(k8s.InitOptions{
		KubeconfigPath: *kubeconfig,
	})
	if err != nil {
		log.Fatalf("Failed to initialize K8s client: %v", err)
	}

	if kubepath := k8s.GetKubeconfigPath(); kubepath != "" {
		log.Printf("Using kubeconfig: %s", kubepath)
	} else {
		log.Printf("Using in-cluster config")
	}

	// Initialize change history (before resource cache so it can receive events)
	historyPath := ""
	if *persistHistory {
		homeDir, _ := os.UserHomeDir()
		historyPath = homeDir + "/.skyhook-explorer/history.jsonl"
	}
	k8s.InitChangeHistory(*historyLimit, historyPath)
	if *persistHistory {
		log.Printf("Change history persistence enabled: %s", historyPath)
	}

	// Initialize resource cache (typed informers for core resources)
	if err := k8s.InitResourceCache(); err != nil {
		log.Fatalf("Failed to initialize resource cache: %v", err)
	}

	log.Printf("Resource cache initialized with %d resources", k8s.GetResourceCache().GetResourceCount())

	// Initialize resource discovery (for CRD support)
	if err := k8s.InitResourceDiscovery(); err != nil {
		log.Printf("Warning: Failed to initialize resource discovery: %v", err)
	}

	// Initialize dynamic resource cache (for CRDs)
	if err := k8s.InitDynamicResourceCache(); err != nil {
		log.Printf("Warning: Failed to initialize dynamic resource cache: %v", err)
	}

	// Initialize Helm client
	if err := helm.Initialize(k8s.GetKubeconfigPath()); err != nil {
		log.Printf("Warning: Failed to initialize Helm client: %v", err)
	}

	// Create and start server
	cfg := server.Config{
		Port:       *port,
		DevMode:    *devMode,
		StaticFS:   static.FS,
		StaticRoot: "dist",
	}

	srv := server.New(cfg)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		srv.Stop()
		if cache := k8s.GetResourceCache(); cache != nil {
			cache.Stop()
		}
		if dynCache := k8s.GetDynamicResourceCache(); dynCache != nil {
			dynCache.Stop()
		}
		os.Exit(0)
	}()

	// Open browser unless disabled
	if !*noBrowser {
		url := fmt.Sprintf("http://localhost:%d", *port)
		if *namespace != "" {
			url += fmt.Sprintf("?namespace=%s", *namespace)
		}
		go openBrowser(url)
	}

	// Start server (blocks)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Cannot open browser on %s, please open manually: %s", runtime.GOOS, url)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
		log.Printf("Please open manually: %s", url)
	}
}
