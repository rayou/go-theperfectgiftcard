# go-theperfectgiftcard

[![GoDoc](https://godoc.org/github.com/rayou/go-theperfectgiftcard?status.svg)](https://godoc.org/github.com/rayou/go-theperfectgiftcard)
[![Go Report Card](https://goreportcard.com/badge/github.com/rayou/go-theperfectgiftcard)](https://goreportcard.com/report/github.com/rayou/go-theperfectgiftcard)
[![Coverage Status](https://coveralls.io/repos/github/rayou/go-theperfectgiftcard/badge.svg?branch=master)](https://coveralls.io/github/rayou/go-theperfectgiftcard?branch=master)

**go-theperfectgiftcard** is a go package that parses [The Perfect Gift Card](https://giftcards.indue.com.au/theperfectgiftcard/) website for your gift card summary and transaction history. 

To check your card summary and transaction history, The Perfect Gift Card HTTP endpoint requires:

1. A Gift Card Number.
2. A Gift Card Pin that is encrypted by an RSA public key.
3. A set of headers for ASP.Net server.

This package takes your card number and pin, builds headers that are required by the server, encrypts your pin by an RSA public key, then log in to The Perfect Gift Card website and parse the data for you.


## Install

```bash
go get github.com/ray.ou/go-theperfectgiftcard
```

## Example

```go
package main

import (
	"log"

	"github.com/rayou/go-theperfectgiftcard"
)

func main() {
	cardNo := "1234567890" // change to your card no
	pin := "0000"          // change to your card pin

	client, err := theperfectgiftcard.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	card, resp, err := client.GetCard(cardNo, pin)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Card No: %v", card.CardNo)
	log.Printf("Available Balance: %v", card.AvailableBalance)
	log.Printf("HTTP Status Code: %v", resp.StatusCode)
}
```

## Contributing

PRs are welcome.

## Author

Ray Ou - yuhung.ou@live.com

## License

MIT.
