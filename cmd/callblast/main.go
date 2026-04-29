package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/rayancheca/callblast/internal/analysis"
	"github.com/rayancheca/callblast/internal/server"
)

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default: // windows
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	_ = cmd.Start()
}

func main() {
	port := flag.Int("port", 7332, "HTTP server port")
	staticDir := flag.String("static", "web/dist", "Directory to serve frontend from")
	demo := flag.Bool("demo", false, "Open browser automatically after server starts")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "callblast — PR blast-radius analyzer\n\n")
		fmt.Fprintf(os.Stderr, "Usage: callblast [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nThen open http://localhost:<port> in your browser.\n")
	}
	flag.Parse()

	analyzer := func(ctx context.Context, req server.AnalysisRequest, events chan<- server.GraphEvent) {
		analysis.RunAnalysis(ctx, req, events)
	}

	srv := server.New(*port, analyzer)

	if *demo {
		url := fmt.Sprintf("http://localhost:%d", *port)
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	if err := srv.Run(*staticDir); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
