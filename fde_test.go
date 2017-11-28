package fde

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

type store map[string]*Transaction

type accounts map[string]bool

func (s store) Get(txid string) (*Transaction, error) {
	t := s[txid]
	if t == nil {
		return nil, nil
	}
	result := &Transaction{}
	*result = *t
	return result, nil
}

func (s store) Append(tt ...*Transaction) ([]string, error) {
	ids := make([]string, len(tt))
	for i, t := range tt {
		t.Id = strconv.Itoa(len(s))
		s[t.Id] = t
		ids[i] = t.Id
	}
	return ids, nil
}

func (a accounts) Exists(nn []string) ([]bool, error) {
	result := make([]bool, len(nn))
	for i, n := range nn {
		result[i] = a[n]
	}
	return result, nil
}

func TestSave(t *testing.T) {
	s := store{}
	r := NewTxsRepository(s, accounts{"1": true, "2": true})
	tx, err := r.Save(&Transaction{
		Debits:  []Entry{{"1", 1}},
		Credits: []Entry{{"2", 1}},
		Date:    time.Now(),
		Memo:    "m",
	})
	check(t, err)
	if len(s) != 1 {
		t.Errorf("Expected 1 but was %v", len(s))
	}
	if tx[0].Id != "0" {
		t.Errorf("Expected 0 but was %v", tx[0].Id)
	}
	tx[0].Memo = "mm"
	_, err = r.Save(tx[0])
	if len(s) != 3 {
		t.Errorf("Expected 3 but was %v", len(s))
	}
	for _, v := range []struct{ value, expect interface{} }{
		{s["0"].Id, "0"},
		{s["0"].Memo, "m"},
		{s["1"].Debits[0].Account, "2"},
		{s["1"].Debits[0].Value, 1},
		{s["1"].Credits[0].Account, "1"},
		{s["1"].Credits[0].Value, 1},
		{s["1"].Memo, "m"},
		{s["1"].Removes, "0"},
		{s["2"].Debits[0].Account, "1"},
		{s["2"].Debits[0].Value, 1},
		{s["2"].Credits[0].Account, "2"},
		{s["2"].Credits[0].Value, 1},
		{s["2"].Memo, "mm"},
	} {
		if fmt.Sprint(v.expect) != fmt.Sprint(v.value) {
			t.Errorf("Expected %v but was %v", v.expect, v.value)
		}
	}
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
