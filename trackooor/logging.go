package trackooor

import (
	"fmt"

	"github.com/Zellic/EVM-trackooor/shared"
)

func logListeningEventsSingle(options shared.TrackooorOptions) {
	// terminal
	fmt.Printf("\n")
	fmt.Printf("Listening to events (single subscription log) on chain ID %v\n", shared.ChainID)
	fmt.Printf("\n")
	// don't output all tracked addresses if too many
	if len(options.FilterAddresses) > 30 {
		fmt.Printf("Filtering by %v contract addresses\n", len(options.FilterAddresses))
	} else {
		fmt.Printf("Filtering by contract addresses (%v): %v\n",
			len(options.FilterAddresses),
			options.FilterAddresses,
		)
	}
	fmt.Printf("Filtering by event topics: %v\n", options.FilterEventTopics)
}

func logListeningBlocks() {
	fmt.Printf("\n")
	fmt.Printf("Listening to blocks mined on chain ID %v\n", shared.ChainID)
	fmt.Printf("\n")
	// don't output all tracked addresses if too many
	if len(shared.Options.FilterAddresses) > 30 {
		fmt.Printf("Filtering by %v addresses\n", len(shared.Options.FilterAddresses))
	} else {
		fmt.Printf(
			"Filtering by addresses (%v): %v\n",
			len(shared.Options.FilterAddresses),
			shared.Options.FilterAddresses,
		)
	}
	fmt.Printf("\n")
}

func logHistoricalEventsSingle() {
	fmt.Printf("\n")
	fmt.Printf("Starting historical events processor - single subscription log\n")
	fmt.Printf("Block range: %v to %v (stepping %v at a time)\n",
		shared.Options.HistoricalOptions.FromBlock,
		shared.Options.HistoricalOptions.ToBlock,
		stepBlocks,
	)
	fmt.Printf("\n")
	// don't output all tracked addresses if too many
	if len(shared.Options.FilterAddresses) > 30 {
		fmt.Printf("Filtering by %v contract addresses\n", len(shared.Options.FilterAddresses))
	} else {
		fmt.Printf("Filtering by contract addresses (%v): %v\n",
			len(shared.Options.FilterAddresses),
			shared.Options.FilterAddresses,
		)
	}
	fmt.Printf("Filtering by event topics (%v): %v\n",
		len(shared.Options.FilterEventTopics),
		shared.Options.FilterEventTopics,
	)
	fmt.Printf("Max RPC requests per second: %v\n", MaxRequestsPerSecond)
	fmt.Printf("\n")
}

func logHistoricalBlocks() {
	fmt.Printf("\n")
	fmt.Printf("Starting historical blocks processor\n")
	fmt.Printf("Block range: %v to %v\n",
		shared.Options.HistoricalOptions.FromBlock,
		shared.Options.HistoricalOptions.ToBlock,
	)
	fmt.Printf("\n")
	// don't output all tracked addresses if too many
	if len(shared.Options.FilterAddresses) > 30 {
		fmt.Printf("Filtering by %v addresses\n", len(shared.Options.FilterAddresses))
	} else {
		fmt.Printf(
			"Filtering by addresses (%v): %v\n",
			len(shared.Options.FilterAddresses),
			shared.Options.FilterAddresses,
		)
	}
	fmt.Printf("Filtering by event topics (%v): %v\n",
		len(shared.Options.FilterEventTopics),
		shared.Options.FilterEventTopics,
	)
	fmt.Printf("Max RPC requests per second: %v\n", MaxRequestsPerSecond)
	fmt.Printf("\n")
}
