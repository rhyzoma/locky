package mocks

import "time"

type FakePasswordHasher struct {
	HashResult    string
	HashErr       error
	CompareResult bool
	HashCalls     int
	CompareCalls  int
}

func (f *FakePasswordHasher) Hash(string) (string, error) {
	f.HashCalls++
	return f.HashResult, f.HashErr
}

func (f *FakePasswordHasher) Compare(string, string) bool {
	f.CompareCalls++
	return f.CompareResult
}

type FakeTokenIssuer struct {
	Token        string
	Err          error
	Calls        int
	LastUserID   string
	LastEmail    string
	LastLifetime time.Duration
}

func (f *FakeTokenIssuer) Sign(userID, email string, lifetime time.Duration) (string, error) {
	f.Calls++
	f.LastUserID = userID
	f.LastEmail = email
	f.LastLifetime = lifetime
	return f.Token, f.Err
}
