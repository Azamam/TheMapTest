package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	WrongCardNumber      = 1
	WrongCardExpiredDate = 2
	WrongCardHolder      = 3
	WrongCVV             = 4
	WrongOrderID         = 5
	WrongAmount          = 6
	CardHasExpired       = 7
	InsufficientFunds    = 8
	UnableToExecute      = 9
)

var errorText = map[int]string{
	WrongCardNumber:      "Wrong card number",
	WrongCardExpiredDate: "Wrong card expired date",
	WrongCardHolder:      "Wrong card holder",
	WrongCVV:             "Wrong CVV",
	WrongOrderID:         "Wrong order id",
	WrongAmount:          "Wrong order Amount",
	CardHasExpired:       "Card Has Expired",
	InsufficientFunds:    "You do not have enough funds on the card",
	UnableToExecute:      "Unable to execute",
}

type ErrorDesc struct {
	ErrorNumber int    `json:"error_number"`
	ErrorMsg    string `json:"error_msg"`
}

type Error struct {
	Error ErrorDesc `json:"error"`
}

type Card struct {
	PAN           string `json:"pan"`
	EMonth        int    `json:"e_month"`
	EYear         int    `json:"e_year"`
	CVV           int    `json:"cvv"`
	Holder        string `json:"holder"`
	amountOnCard  int
	blockedAmount int
}

type Deal struct {
	OrderID string `json:"order_id"`
	Amount  int    `json:"amount"`
	DealID  int    `json:"deal_id"`
}

type Transaction struct {
	MerchantContractID int  `json:"merchant_contract_id"`
	Card               Card `json:"card"`
	Deal               Deal `json:"deal"`
}

var counter = 1
var transactions = make(map[int]Transaction)
var cards = make(map[string]Card)

func checkCardNumber(purportedCC string) bool {
	if purportedCC != "" {
		purportedCC = strings.ReplaceAll(purportedCC, " ", "")
		var sum = 0
		var nDigits = len(purportedCC)
		var parity = nDigits % 2

		for i := 0; i < nDigits; i++ {
			var digit = int(purportedCC[i] - 48)
			if i%2 == parity {
				digit *= 2
				if digit > 9 {
					digit -= 9
				}
			}
			sum += digit
		}
		return sum%10 == 0
	}
	return false
}

func checkCardExpiryDate(eMonth int, eYear int) bool {
	currentDate := time.Now()
	if currentDate.Year() > eYear {
		return false
	} else if int(currentDate.Month()) > eMonth {
		return false
	} else {
		return true
	}
}

func validateData(transaction Transaction) (Error, bool) {
	if checkCardNumber(transaction.Card.PAN) {
		if checkCardExpiryDate(transaction.Card.EMonth, transaction.Card.EYear) {
			if transaction.Card.Holder != "" {
				if transaction.Card.CVV > 99 && transaction.Card.CVV < 1000 {
					if transaction.Deal.OrderID != "" {
						if transaction.Deal.Amount > 0 {
							return Error{}, true
						} else {
							return Error{ErrorDesc{WrongAmount, errorText[WrongAmount]}}, false
						}
					} else {
						return Error{ErrorDesc{WrongOrderID, errorText[WrongOrderID]}}, false
					}
				} else {
					return Error{ErrorDesc{WrongCVV, errorText[WrongCVV]}}, false
				}
			} else {
				return Error{ErrorDesc{WrongCardHolder, errorText[WrongCardHolder]}}, false
			}
		} else {
			if transaction.Card.EMonth > 0 && transaction.Card.EYear > 1970 {
				return Error{ErrorDesc{CardHasExpired, errorText[CardHasExpired]}}, false
			}
			return Error{ErrorDesc{WrongCardExpiredDate, errorText[WrongCardExpiredDate]}}, false
		}
	} else {
		return Error{ErrorDesc{WrongCardNumber, errorText[WrongCardNumber]}}, false
	}

}

func validateDeal(card Card, dealAuth Deal) (Error, bool) {
	if card.amountOnCard >= dealAuth.Amount {
		return Error{}, true
	} else {
		return Error{ErrorDesc{InsufficientFunds, errorText[InsufficientFunds]}}, false
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(http.StatusForbidden)
	}
}

func Block(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(w, "err %q\n", err, err.Error())
		} else {
			var transaction Transaction
			err = json.Unmarshal(body, &transaction)
			if err != nil {
				log.Fatal(w, "Can't unmarshal: ", err.Error())
			} else {
				errJson, ok := validateData(transaction)
				if ok {
					transaction.Deal.DealID = counter
					counter += 1
					transactions[transaction.Deal.DealID] = transaction
					if card, ok := cards[transaction.Card.PAN]; ok {
						card.blockedAmount = transaction.Deal.Amount
						card.amountOnCard -= card.blockedAmount
						cards[card.PAN] = card
						deal := make(map[string]int)
						deal["deal_id"] = transaction.Deal.DealID
						fmt.Println("Block:", cards[transaction.Card.PAN])
						_ = json.NewEncoder(w).Encode(deal)
					}
				} else {
					_ = json.NewEncoder(w).Encode(errJson)
				}
			}
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
	}

}

func Charge(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(w, "err %q\n", err, err.Error())
		} else {
			var deal Deal
			err = json.Unmarshal(body, &deal)
			if err != nil {
				log.Fatal(w, "Can't unmarshal: ", err.Error())
			} else {
				if transact, ok := transactions[deal.DealID]; ok {
					card := cards[transact.Card.PAN]
					errJson, ok := validateDeal(card, transact.Deal)
					if ok {
						card.blockedAmount -= deal.Amount
						cards[card.PAN] = card
						resp := make(map[string]string)
						resp["Operation"] = "Success"
						delete(transactions, deal.DealID)
						fmt.Println("Charge:", cards[transact.Card.PAN])
						_ = json.NewEncoder(w).Encode(resp)
					} else {
						card.amountOnCard += transact.Card.blockedAmount
						card.blockedAmount -= transact.Deal.Amount
						cards[card.PAN] = card
						_ = json.NewEncoder(w).Encode(errJson)
					}
				} else {
					_ = json.NewEncoder(w).Encode(
						Error{ErrorDesc{UnableToExecute, errorText[UnableToExecute]}})
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
	}
}

func handleRequests() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/block", Block)
	http.HandleFunc("/charge", Charge)
	log.Fatal(http.ListenAndServe(":7000", nil))
}

func main() {
	card := Card{
		"4012888888881881",
		9,
		2019,
		100,
		"IVANOV IVAN",
		55555,
		0,
	}
	cards[card.PAN] = card
	handleRequests()
}
