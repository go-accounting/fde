package fde

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type Transaction struct {
	Id      string    `json:"id"`
	Debits  []Entry   `json:"debits"`
	Credits []Entry   `json:"credits"`
	Date    time.Time `json:"date"`
	Memo    string    `json:"memo"`
	Tags    []string  `json:"tags"`
	User    string    `json:"user"`
	AsOf    time.Time `json:"timestamp"`
	Removes string    `json:"removes"`
	// AccountsKeysAsString []string  `json:"-"`
	// Moment               int64     `datastore:"-" json:"-"`
}

type Entry struct {
	Account string  `json:"account"`
	Value   float64 `json:"value"`
}

type TxsRepository struct {
	s  Store
	ar AccountsRepository
}

type Store interface {
	Get(txid string) (*Transaction, error)
	Append(*Transaction) (string, error)
}

type AccountsRepository interface {
	Indexes([]string) ([]int, error)
}

func NewTxsRepository(s Store, ar AccountsRepository) *TxsRepository {
	return &TxsRepository{s, ar}
}

func (tr *TxsRepository) Save(t *Transaction) (*Transaction, error) {
	if msg := t.ValidationMessage(tr); msg != "" {
		return nil, fmt.Errorf(msg)
	}
	if t.Id != "" {
		_, err := tr.Delete(t.Id)
		if err != nil {
			return nil, err
		}
	}
	result := &Transaction{}
	*result = *t
	var err error
	result.Id, err = tr.s.Append(t)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (tr *TxsRepository) Delete(txid string) (*Transaction, error) {
	tx, err := tr.s.Get(txid)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, fmt.Errorf("Transaction not found")
	}
	deb := make([]Entry, len(tx.Credits))
	cre := make([]Entry, len(tx.Debits))
	for i, e := range tx.Debits {
		cre[i] = Entry{Account: e.Account, Value: e.Value}
	}
	for i, e := range tx.Credits {
		deb[i] = Entry{Account: e.Account, Value: e.Value}
	}
	tx.Debits = deb
	tx.Credits = cre
	tx.Removes = txid
	tx.Id, err = tr.s.Append(tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (transaction *Transaction) ValidationMessage(tr *TxsRepository) string {
	if len(transaction.Debits) == 0 {
		return "At least one debit must be informed"
	}
	if len(transaction.Credits) == 0 {
		return "At least one credit must be informed"
	}
	if transaction.Date.IsZero() {
		return "The date must be informed"
	}
	if len(strings.TrimSpace(transaction.Memo)) == 0 {
		return "The memo must be informed"
	}
	ev := func(arr []Entry) (string, float64) {
		sum := 0.0
		for _, e := range arr {
			if m := e.ValidationMessage(tr); len(m) > 0 {
				return m, 0.0
			}
			sum += e.Value
		}
		return "", sum
	}
	var debitsSum, creditsSum float64
	var m string
	if m, debitsSum = ev(transaction.Debits); len(m) > 0 {
		return m
	}
	if m, creditsSum = ev(transaction.Credits); len(m) > 0 {
		return m
	}
	if math.Trunc(debitsSum*100+0.5) != math.Trunc(creditsSum*100+0.5) {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (entry *Entry) ValidationMessage(tr *TxsRepository) string {
	if entry.Account == "" {
		return "The account must be informed for each entry"
	}
	idx, err := tr.ar.Indexes([]string{entry.Account})
	if err != nil {
		return err.Error()
	}
	if idx[0] == -1 {
		return "Account not found"
	}
	// The following code is not necessary if using mcesar.io/coa because we can use the
	// method Indexes informing the tag surrounded the method by a adapter
	// if !collections.Contains(account.Tags, "analytic") {
	// 	return "The account must be analytic"
	// }
	return ""
}
