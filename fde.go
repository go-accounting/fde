package fde

import (
	"fmt"
	"strings"
	"time"
)

type Transaction struct {
	Id      string    `json:"_id"`
	Debits  Entries   `json:"debits"`
	Credits Entries   `json:"credits"`
	Date    time.Time `json:"date"`
	Memo    string    `json:"memo"`
	Tags    []string  `json:"tags"`
	User    string    `json:"user"`
	AsOf    time.Time `json:"timestamp"`
	Removes string    `json:"removes"`
}

type Entry struct {
	Account string `json:"account"`
	Value   int64  `json:"value"`
}

type TxsRepository struct {
	s  Store
	ar AccountsRepository
}

type Store interface {
	Get(txid string) (*Transaction, error)
	Append(...*Transaction) ([]string, error)
}

type AccountsRepository interface {
	Exists([]string) ([]bool, error)
}

type Entries []Entry

func NewTxsRepository(s Store, ar AccountsRepository) *TxsRepository {
	return &TxsRepository{s, ar}
}

func (tr *TxsRepository) Save(tt ...*Transaction) ([]*Transaction, error) {
	rr := make([]Transaction, len(tt))
	result := make([]*Transaction, len(tt))
	for i, t := range tt {
		if msg := t.ValidationMessage(tr); msg != "" {
			return nil, fmt.Errorf(msg)
		}
		if t.Id != "" {
			_, err := tr.Delete(t.Id)
			if err != nil {
				return nil, err
			}
		}
		result[i] = &rr[i]
		*result[i] = *t
	}
	ids, err := tr.s.Append(tt...)
	if err != nil {
		return nil, err
	}
	for i, id := range ids {
		result[i].Id = id
	}
	return result, nil
}

func (tr *TxsRepository) Get(txid string) (*Transaction, error) {
	return tr.s.Get(txid)
}

func (tr *TxsRepository) Delete(txid string) (*Transaction, error) {
	// TODO: prevent multiple deletes on same transaction
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
	ids, err := tr.s.Append(tx)
	if err != nil {
		return nil, err
	}
	tx.Id = ids[0]
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
	if m := transaction.Debits.ValidationMessage(tr); len(m) > 0 {
		return m
	}
	if m := transaction.Credits.ValidationMessage(tr); len(m) > 0 {
		return m
	}
	if transaction.Debits.sum() != transaction.Credits.sum() {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (ee Entries) sum() int64 {
	result := int64(0)
	for _, e := range ee {
		result += e.Value
	}
	return result
}

func (ee Entries) ValidationMessage(tr *TxsRepository) string {
	ids := make([]string, len(ee))
	for i, entry := range ee {
		if entry.Account == "" {
			return "The account must be informed for each entry"
		}
		ids[i] = entry.Account
	}
	oks, err := tr.ar.Exists(ids)
	if err != nil {
		return err.Error()
	}
	for i, ok := range oks {
		if !ok {
			return fmt.Sprintf("The account '%v' was not found or it does not have the correct type",
				ids[i])
		}
	}
	// The following code is not necessary if using github.com/go-accounting/coa because we can use the
	// method Indexes informing the tag surrounded the method by a adapter
	// if !collections.Contains(account.Tags, "detail") {
	// 	return "The account type must be detail"
	// }
	return ""
}
