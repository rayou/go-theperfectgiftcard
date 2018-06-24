package theperfectgiftcard

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

const (
	baseURL  = "https://giftcards.indue.com.au/theperfectgiftcard/"
	modulus  = "D4229AD35AE3FE60E192948079DFEE523E018CE5931E6FF68A70C20A2A91D80FF09604DE9F4100C5B91A8433712428B35F3CC6C4CA814715BE470D811E73BE497788CA38494CADAF4825E78A508FAB023F65FC4722306FE7ECF1AC41C19AE5C4EFD3ACFE99EE08B60794EC19D57EA0E3556EE53F8DAECAB67DB47AFBC0F856AD"
	exp      = "010001"
	randomNo = "7464663221746466322174646632217464663221"
)

var body = map[string]string{
	"__VIEWSTATE":          "/wEPDwUJODY3MDYzNzgzD2QWAgIDD2QWBmYPFgIeB1Zpc2libGVoZAIDDw9kFgIeB29uY2xpY2sFFXJldHVybiBnZXRwYXNzd29yZCgpO2QCBg8WAh8AaGRkXcXstETbLhPK3PqD3TU7Io+Xaw4=",
	"__VIEWSTATEGENERATOR": "5898F960",
	"__EVENTVALIDATION":    "/wEWBgKuoL7TCwLi0uqnCgK1qbSRCwKFoZPNAwLQvbH7BAL8yZzMCZCDAp2k+4wbYm+6XCicWxiA53iU",
	"cmdLogin":             " Log+in ",
	"hdnrandomnumber":      randomNo,
}

// Response is a simplied HTTP response. This wraps colly.Response returned from
// the The Perfect Gift Card website.
type Response struct {
	*colly.Response
}

// Transaction contains the detail of a transaction.
type Transaction struct {
	Date        string
	Details     string
	Description string
	Amount      string
	Balance     string
}

// Card contains card summary and transaction history.
type Card struct {
	CardNo           string
	AccountNo        string
	LoadsToDate      string
	PurchasesToDate  string
	AvailableBalance string
	PurchasedDate    string
	ExpiryDate       string
	Transactions     []Transaction
}

// Client manages the communicate with the The Perfect Gift Card website.
type Client struct {
	BaseURL   *url.URL
	publicKey rsa.PublicKey
}

// NewClient creates a ThePerfectGiftCard cilent.
func NewClient() (*Client, error) {
	baseURL, _ := url.Parse(baseURL)
	publicKey, _ := makePublicKey(modulus, exp)
	return &Client{
		BaseURL:   baseURL,
		publicKey: publicKey,
	}, nil
}

// GetCard returns a Card struct with card summary and transaction history, a
// Response which contains a simplied HTTP response, and an error. Takes card
// number and pin which will be used to login to The Perfect Gift Card website.
func (c *Client) GetCard(cardNo string, pin string) (*Card, *Response, error) {
	card := &Card{}
	response := &Response{}
	var err error
	encryptedPin, err := encryptPinWithPublicKey(pin, c.publicKey)
	if err != nil {
		return card, response, err
	}

	co := colly.NewCollector()

	co.OnHTML("#ctl00_DefaultContent_lblMembershipNumber", func(e *colly.HTMLElement) {
		card.CardNo = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblAccountNumber", func(e *colly.HTMLElement) {
		card.AccountNo = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblcardvalue", func(e *colly.HTMLElement) {
		card.LoadsToDate = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblpurchasestodate", func(e *colly.HTMLElement) {
		card.PurchasesToDate = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblavailablebalance", func(e *colly.HTMLElement) {
		card.AvailableBalance = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblCardPurchasedDate", func(e *colly.HTMLElement) {
		card.PurchasedDate = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#ctl00_DefaultContent_lblCardExpiryDate", func(e *colly.HTMLElement) {
		card.ExpiryDate = strings.TrimSpace(e.Text)
	})

	co.OnHTML("#htmltdErrorDescription", func(e *colly.HTMLElement) {
		err = errors.New(e.Text)
		response.StatusCode = http.StatusUnauthorized
	})

	co.OnHTML(".content-error h3", func(e *colly.HTMLElement) {
		err = errors.New("internal server error")
		response.StatusCode = http.StatusInternalServerError
	})

	co.OnHTML("#dgPointsStatement tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(i int, el *colly.HTMLElement) {
			// skip title row
			if i == 0 {
				return
			}
			transaction := Transaction{
				Date:        strings.TrimSpace(el.ChildText("td:nth-of-type(1)")),
				Details:     strings.TrimSpace(el.ChildText("td:nth-of-type(2)")),
				Description: strings.TrimSpace(el.ChildText("td:nth-of-type(3)")),
				Amount:      strings.TrimSpace(el.ChildText("td:nth-of-type(4)")),
				Balance:     strings.TrimSpace(el.ChildText("td:nth-of-type(5)")),
			}
			card.Transactions = append(card.Transactions, transaction)
		})
	})

	co.OnResponse(func(r *colly.Response) {
		response = &Response{Response: r}
	})

	co.OnError(func(r *colly.Response, e error) {
		err = e
		response.Response = r
	})

	body["txtCardNumber"] = cardNo
	body["hdnrequest"] = fmt.Sprintf("%x", encryptedPin)
	co.Post(c.BaseURL.String(), body)
	return card, response, err
}

func makePublicKey(modulus string, exp string) (rsa.PublicKey, error) {
	n := new(big.Int)
	n, ok := n.SetString(modulus, 16)
	if !ok {
		return rsa.PublicKey{}, errors.New("invalid modulus")
	}
	e, err := strconv.ParseInt(exp, 16, 0)
	if err != nil {
		return rsa.PublicKey{}, err
	}
	return rsa.PublicKey{
		N: n,
		E: int(e),
	}, nil
}

func encryptPinWithPublicKey(pin string, publicKey rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, &publicKey, []byte(fmt.Sprintf("%s|%s", pin, randomNo)))
}
