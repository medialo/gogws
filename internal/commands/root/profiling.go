package root

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"

	"github.com/spf13/cobra"
)

var (
	allProfile       bool
	traceProfile     bool
	cPUProfile       bool
	mEMProfile       bool
	blockProfile     bool
	traceProfileFile = "cpu.prof"
	cPUProfileFile   = "cpu.prof"
	mEMProfileFile   = "mem.prof"
	blkProfileFile   = "block.prof"
)

var onStopProfiling func()
var profilingOnce sync.Once

func profilingLog() *slog.Logger {
	return slog.With("component", "PROFILING")
}

func applyProfilingFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().BoolVar(&allProfile, "all-profile", false, "enable all profiling (CPU, Trace, memory, blocking)")
	rootCmd.PersistentFlags().BoolVar(&traceProfile, "trace-profile", false, "enable trace profiling")
	rootCmd.PersistentFlags().BoolVar(&cPUProfile, "cpu-profile", false, "enable CPU profiling")
	rootCmd.PersistentFlags().BoolVar(&mEMProfile, "mem-profile", false, "enable memory profiling")
	rootCmd.PersistentFlags().BoolVar(&blockProfile, "block-profile", false, "enable blocking profile")
	rootCmd.PersistentFlags().StringVar(&traceProfileFile, "trace-profile-file", "trace.out", "file to write trace profile to")
	rootCmd.PersistentFlags().StringVar(&cPUProfileFile, "cpu-profile-file", "cpu.prof", "file to write CPU profile to")
	rootCmd.PersistentFlags().StringVar(&mEMProfileFile, "mem-profile-file", "mem.prof", "file to write memory profile to")
	rootCmd.PersistentFlags().StringVar(&blkProfileFile, "block-profile-file", "block.prof", "file to write blocking profile to")

	// make all flags hidden
	rootCmd.PersistentFlags().MarkHidden("all-profile")
	rootCmd.PersistentFlags().MarkHidden("trace-profile")
	rootCmd.PersistentFlags().MarkHidden("cpu-profile")
	rootCmd.PersistentFlags().MarkHidden("mem-profile")
	rootCmd.PersistentFlags().MarkHidden("block-profile")
	rootCmd.PersistentFlags().MarkHidden("trace-profile-file")
	rootCmd.PersistentFlags().MarkHidden("cpu-profile-file")
	rootCmd.PersistentFlags().MarkHidden("mem-profile-file")
	rootCmd.PersistentFlags().MarkHidden("block-profile-file")
}

func StopProfiling() {
	if onStopProfiling != nil {
		profilingLog().Debug("Stopping profiling...")
		profilingOnce.Do(onStopProfiling)
	}
}

func profilingInit() func() {

	if allProfile {
		profilingLog().Debug("All profiling requested, enabling all profiles")
		cPUProfile = true
		traceProfile = true
		mEMProfile = true
		blockProfile = true
	}

	if !cPUProfile && !mEMProfile && !blockProfile && !traceProfile {
		profilingLog().Debug("No profiling profile requested")
		return nil
	}

	profilingLog().Debug("Initializing profiling...", "CPUProfile", cPUProfile, "MEMProfile", mEMProfile, "BlockProfile", blockProfile, "TraceProfile", traceProfile)
	var doOnStop []func()

	stop := func() {
		for _, d := range doOnStop {
			if d != nil {
				d()
			}
		}
	}

	// CPU profile
	if cPUProfile {
		f, err := os.Create(cPUProfileFile)
		if err != nil {
			fmt.Println("could not create cpu profile:", err)
			return stop
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Println("could not start cpu profiling:", err)
			return stop
		}
		doOnStop = append(doOnStop, func() {
			pprof.StopCPUProfile()
			_ = f.Close()
			fmt.Println("✓ cpu profile saved →", cPUProfileFile)
		})
	}

	// Memory (heap) profile
	if mEMProfile {
		f, err := os.Create(mEMProfileFile)
		if err != nil {
			fmt.Println("could not create mem profile:", err)
			return stop
		}
		doOnStop = append(doOnStop, func() {
			runtime.GC()
			_ = pprof.WriteHeapProfile(f)
			_ = f.Close()
			fmt.Println("✓ mem profile saved →", mEMProfileFile)
		})
	}

	// Block profile
	if blockProfile {
		runtime.SetBlockProfileRate(1) // capture TOUS les blocages
		f, err := os.Create(blkProfileFile)
		if err != nil {
			fmt.Println("could not create block profile:", err)
			return stop
		}
		doOnStop = append(doOnStop, func() {
			_ = pprof.Lookup("block").WriteTo(f, 0)
			_ = f.Close()
			runtime.SetBlockProfileRate(0)
			fmt.Println("✓ block profile saved →", blkProfileFile)
		})
	}

	// command to analyse : go tool trace trace.out
	if traceProfile {
		f, _ := os.Create(traceProfileFile)
		trace.Start(f)
		doOnStop = append(doOnStop, func() {
			trace.Stop()
			f.Close()
			fmt.Println("✓ trace saved → trace.out")
		})
	}

	return stop
}
