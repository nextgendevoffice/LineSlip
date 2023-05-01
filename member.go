package main

import "sync"

type MemberSystem struct {
	members map[string]bool
	mu      sync.Mutex
}

func NewMemberSystem() *MemberSystem {
	return &MemberSystem{
		members: make(map[string]bool),
	}
}

func (ms *MemberSystem) AddMember(userID string) {
	ms.mu.Lock()
	ms.members[userID] = true
	ms.mu.Unlock()
}

func (ms *MemberSystem) IsMember(userID string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.members[userID]
}
