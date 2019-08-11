package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type (
	ErrorDesc struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}

	Error struct {
		Error ErrorDesc `json:"error"`
	}

	Card struct {
		PAN           string `json:"pan"`
		EMonth        int    `json:"e_month"`
		EYear         int    `json:"e_year"`
		CVV           int    `json:"cvv"`
		Holder        string `json:"holder"`
		amountOnCard  int
		blockedAmount int
	}

	Deal struct {
		OrderID string `json:"order_id"`
		Amount  int    `json:"amount"`
		DealID  int    `json:"deal_id"`
	}

	Transaction struct {
		MerchantContractID int  `json:"merchant_contract_id"`
		Card               Card `json:"card"`
		Deal               Deal `json:"deal"`
	}
)

var (
	WrongCardNumber      = myOnlyError(1, "Wrong card number")
	WrongCardExpiredDate = myOnlyError(2, "Wrong card expired date")
	WrongCardHolder      = myOnlyError(3, "Wrong card holder")
	WrongCVV             = myOnlyError(4, "Wrong CVV")
	WrongOrderID         = myOnlyError(5, "Wrong order id")
	WrongAmount          = myOnlyError(6, "Wrong order Amount")
	CardHasExpired       = myOnlyError(7, "Card Has Expired")
	InsufficientFunds    = myOnlyError(8, "You do not have enough funds on the card")
	UnableToExecute      = myOnlyError(9, "Unable to execute")

	ch                 = make(chan int, 10)
	counterTransaction = 1
	transactions       = make(map[int]Transaction)
	cards              = make(map[string]Card)
)

func myOnlyError(code int, errorMsg string) Error {
	return Error{ErrorDesc{code, errorMsg}}
}

func isValidCardNumber(purportedCC string) bool {
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

func isValidCardExpiryDate(eMonth int, eYear int) bool {
	currentDate := time.Now()
	if currentDate.Year() > eYear {
		return false
	} else if int(currentDate.Month()) > eMonth {
		return false
	} else {
		return true
	}
}

func (transaction *Transaction) validate() (Error, bool) {
	if !isValidCardNumber(transaction.Card.PAN) {
		return WrongCardNumber, false
	}
	if !isValidCardExpiryDate(transaction.Card.EMonth, transaction.Card.EYear) {
		if transaction.Card.EMonth > 0 && transaction.Card.EYear > 1970 {
			return CardHasExpired, false
		}
		return WrongCardExpiredDate, false
	}
	if transaction.Card.Holder == "" || transaction.Card.Holder != cards[transaction.Card.PAN].Holder {
		return WrongCardHolder, false
	}
	if transaction.Card.CVV < 99 &&
		transaction.Card.CVV > 1000 &&
		cards[transaction.Card.PAN].CVV == transaction.Card.CVV {
		return WrongCVV, false
	}
	if transaction.Deal.OrderID == "" {
		return WrongOrderID, false
	}
	if transaction.Deal.Amount <= 0 {
		return WrongAmount, false
	}
	return Error{}, true
}

func (dealAuth *Deal) validate(card Card) (Error, bool) {
	if card.amountOnCard >= dealAuth.Amount {
		return Error{}, true
	} else {
		return InsufficientFunds, false
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.WriteHeader(http.StatusForbidden)
	}
	go requestToYa()
	ch <- 0
}

func Block(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(w, "err %q\n", err, err.Error())
		}
		var transaction Transaction
		err = json.Unmarshal(body, &transaction)
		if err != nil {
			log.Fatal(w, "Can't unmarshal: ", err.Error())
		}
		if errJson, ok := transaction.validate(); ok {
			transaction.Deal.DealID = counterTransaction
			counterTransaction += 1
			transactions[transaction.Deal.DealID] = transaction
			if card, ok := cards[transaction.Card.PAN]; ok {
				if errJson, ok = transaction.Deal.validate(card); ok {
					card.blockedAmount += transaction.Deal.Amount
					card.amountOnCard -= transaction.Deal.Amount
					cards[card.PAN] = card
					deal := make(map[string]int)
					deal["deal_id"] = transaction.Deal.DealID
					log.Println("Block:", cards[transaction.Card.PAN])
					_ = json.NewEncoder(w).Encode(deal)
				} else {
					delete(transactions, transaction.Deal.DealID)
					_ = json.NewEncoder(w).Encode(errJson)
				}
			}
		} else {
			_ = json.NewEncoder(w).Encode(errJson)
		}

	} else {
		w.WriteHeader(http.StatusForbidden)
	}
	go requestToYa()
	ch <- 0
}

func Charge(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(w, "err %q\n", err, err.Error())
		}
		var deal Deal
		err = json.Unmarshal(body, &deal)
		if err != nil {
			log.Fatal(w, "Can't unmarshal: ", err.Error())
		}
		if transact, ok := transactions[deal.DealID]; ok {
			card := cards[transact.Card.PAN]
			card.blockedAmount -= deal.Amount
			cards[card.PAN] = card
			resp := make(map[string]string)
			resp["Operation"] = "Success"
			delete(transactions, deal.DealID)
			log.Println("Charge:", cards[transact.Card.PAN])
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			_ = json.NewEncoder(w).Encode(UnableToExecute)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	go requestToYa()
	ch <- 0
}

func handleRequests() {
	go http.HandleFunc("/", Index)
	go http.HandleFunc("/block", Block)
	go http.HandleFunc("/charge", Charge)
	log.Fatal(http.ListenAndServe(":7000", nil))
}

func requestToYa() {
	r, err := http.Get("https://ya.ru/")
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("ya.ru status code:", r.StatusCode)
	<-ch
}

func main() {
	internalCards := []Card{
		{"4012888888881881", 9, 2019, 100, "IVANOV IVAN", 200, 0},
		{"4539068477119696", 9, 2019, 100, "IVANOV IVAN", 200, 0},
		{"5192679737272623", 9, 2019, 100, "IVANOV IVAN", 0, 0},
		{"5420893164982661", 9, 2019, 100, "IVANOV IVAN", 0, 0},
		{"344681483420255", 9, 2019, 100, "IVANOV IVAN", 1000, 0},
		{"341487100962668", 9, 2019, 100, "IVANOV IVAN", 1000, 0},
		{"6011755772471507", 9, 2019, 100, "IVANOV IVAN", 68000, 0},
		{"6011937144761860", 9, 2019, 100, "IVANOV IVAN", 68315, 0},
	}
	for _, card := range internalCards {
		cards[card.PAN] = card
	}
	handleRequests()
}
