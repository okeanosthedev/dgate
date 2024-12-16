package main

import (
	"flag"
	"fmt"
	"os"

	"go.minekube.com/gate/pkg/edition/java/lite/blacklist"
)

func main() {
	blacklistFile := flag.String("file", "ip_blacklist.json", "Path to the global blacklist JSON file")
	add := flag.String("add", "", "IP address to add to the blacklist")
	remove := flag.String("remove", "", "IP address to remove from the blacklist")
	list := flag.Bool("list", false, "List all blacklisted IPs")
	duration := flag.Duration("duration", 0, "Duration for the blacklist entry (e.g., 24h, 7d, 30d). Use 0 for unlimited.")
	flag.Parse()

	bl, err := blacklist.NewBlacklist(*blacklistFile)
	if err != nil {
		fmt.Printf("Error initializing global blacklist: %v\n", err)
		os.Exit(1)
	}

	if *add != "" {
		err = bl.Add(*add, *duration)
		if err != nil {
			fmt.Printf("Error adding IP to blacklist: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added %s to the blacklist", *add)
		if *duration > 0 {
			fmt.Printf(" for %s", *duration)
		}
		fmt.Println()
	}

	if *remove != "" {
		err = bl.Remove(*remove)
		if err != nil {
			fmt.Printf("Error removing IP from blacklist: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %s from the blacklist\n", *remove)
	}

	if *list {
		fmt.Println("Global Blacklisted IPs:")
		for _, ip := range bl.GetIPs() {
			fmt.Println(ip)
		}
	}
}