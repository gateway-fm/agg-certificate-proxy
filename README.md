# AggLayer Certificate Proxy
A small proxy to provide the following functionality:
- Delay outgoing AggLayer certificates
- Provide statistics on outgoing AggLayer certificates
- Allow a kill-switch type functionality to prevent outbound bridging in case of network emergency

## Work In Progress:

### Cert exit info
	// A certificate can have multiple bridge exits
	if len(cert.BridgeExits) == 0 {
		fmt.Println("Certificate contains no bridge exits.")
	} else {
		fmt.Printf("Found %d bridge exit(s):\n", len(cert.BridgeExits))
		for i, exit := range cert.BridgeExits {
			fmt.Printf(" - Exit %d: Amount = %s, Destination = %s\n",
				i+1,
				exit.Amount.String(), // Here is the amount
				exit.DestinationAddress.Hex(),
			)
		}
	}

## TODO
- add 2x new flags on startup to be stored in a new table 'credentials', they will be used to authenticate the APIs and should be passed on query string as ?key=
		- mgmt_api_key
		- stats_api_key
- add repo to aikido and security scan it
