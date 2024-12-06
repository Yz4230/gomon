/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

var (
	interval time.Duration
	output   string
	ifaces   []string
	silent   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gomon",
	Short: "Network interface traffic logger",
	Run: func(cmd *cobra.Command, args []string) {
		file, err := os.Create(output)
		if err != nil {
			log.Fatalf("failed to create output file: %v", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		writer.Write([]string{"time", "if_idx", "if_name", "rx_bytes", "tx_bytes"})
		collectTicker := time.NewTicker(interval)

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case <-collectTicker.C:
				ts := time.Now().UTC().Format(time.RFC3339Nano)
				links, err := netlink.LinkList()
				if err != nil {
					log.Fatalf("failed to list network interfaces: %v", err)
				}
				for _, link := range links {
					attr := link.Attrs()
					stat := attr.Statistics
					if len(ifaces) > 0 && slices.Contains(ifaces, attr.Name) {
						continue
					}
					writer.Write([]string{
						ts,
						fmt.Sprint(attr.Index),
						attr.Name,
						fmt.Sprint(stat.RxBytes),
						fmt.Sprint(stat.TxBytes),
					})
					if !silent {
						fmt.Printf("[%s] %s: RX %s, TX %s\n",
							time.Now().Format(time.TimeOnly),
							attr.Name,
							humanizeSize(stat.RxBytes),
							humanizeSize(stat.TxBytes),
						)
					}
				}
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().DurationVarP(&interval, "interval", "i", time.Second, "Polling interval (e.g., 10ms, 1s)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: {timestamp}.csv)")
	rootCmd.Flags().StringArrayVarP(&ifaces, "iface", "f", []string{}, "Network interface(s) to monitor")
	rootCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Silent mode")

	if output == "" {
		output = fmt.Sprintf("%s.csv", time.Now().UTC().Format(time.RFC3339))
	}
}

func humanizeSize(size uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}

	var i int
	for i = 0; size >= 1024 && i < len(units); i++ {
		size /= 1024
	}

	return fmt.Sprintf("%d %s", size, units[i])
}
