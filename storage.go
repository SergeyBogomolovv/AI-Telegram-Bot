package main

import (
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type ChatStorage interface {
	Chat(chatID int64) (Chat, bool)
	CreateChat(chatID int64) Chat
	GetOrCreateChat(chatID int64) Chat
}

type chatStorage struct {
	llm   llms.Model
	chats map[int64]Chat
	mu    sync.RWMutex
}

func NewChatStorage(llm llms.Model) ChatStorage {
	return &chatStorage{
		llm:   llm,
		chats: make(map[int64]Chat),
	}
}

func (s *chatStorage) Chat(chatID int64) (Chat, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chat, exists := s.chats[chatID]
	return chat, exists
}

func (s *chatStorage) CreateChat(chatID int64) Chat {
	s.mu.Lock()
	defer s.mu.Unlock()
	chat := NewChat(s.llm)
	s.chats[chatID] = chat
	return chat
}

func (s *chatStorage) GetOrCreateChat(chatID int64) Chat {
	chat, exists := s.Chat(chatID)
	if exists {
		return chat
	}
	return s.CreateChat(chatID)
}

type StateStorage interface {
	IsWaitingForRole(chatID int64) bool
	SetWaitingForRole(chatID int64, state bool)
}

type stateStorage struct {
	waitingForRole map[int64]bool
	mu             sync.RWMutex
}

func NewStateStorage() StateStorage {
	return &stateStorage{
		waitingForRole: make(map[int64]bool),
	}
}

func (s *stateStorage) IsWaitingForRole(chatID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.waitingForRole[chatID]
}

func (s *stateStorage) SetWaitingForRole(chatID int64, state bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitingForRole[chatID] = state
}
