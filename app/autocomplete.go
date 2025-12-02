package main

type trieNode struct {
	char     string
	children map[string]*trieNode
}

type stackItem struct {
	trie        *trieNode
	currentWord string
}

func (t *trieNode) insert(word string) {
	current := t
	for _, c := range word {
		letter := string(c)
		n, ok := current.children[letter]
		if !ok {
			newNode := &trieNode{
				char:     letter,
				children: make(map[string]*trieNode),
			}
			current.children[letter] = newNode
			current = newNode
			continue
		}
		current = n
	}
	current.children["*"] = nil
}

func (t *trieNode) search(word string) bool {
	current := t
	for _, c := range word {
		letter := string(c)
		n, ok := current.children[letter]
		if !ok {
			return false
		}
		current = n
	}
	if _, ok := current.children["*"]; !ok {
		return false
	}
	return true
}

func (t *trieNode) prefixSearch(word string) []string {
	current := t
	for _, c := range word {
		letter := string(c)
		n, ok := current.children[letter]
		if !ok {
			return nil
		}
		current = n
	}
	wordsFound := findWordMatches(current, word)
	return wordsFound
}


func findWordMatches(t *trieNode, word string) []string {
	wordsFound := []string{}
	stack := getTrieNodes(t, word)
	currentWord := word
	for len(stack) > 0 {
		currentNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if currentNode.trie == nil {
			wordsFound = append(wordsFound, currentNode.currentWord)
			if (len(stack) > 0) {
				currentWord = stack[len(stack) - 1].currentWord
			}
			continue
		}
		currentWord += currentNode.trie.char
		stack = append(stack, getTrieNodes(currentNode.trie, currentWord)...)
	}
	return wordsFound
}

func getTrieNodes(t *trieNode, word string) []stackItem{
	stringKeys := []stackItem{}
	for _, val := range t.children {
		stringKeys = append(stringKeys, stackItem{trie: val, currentWord: word})

	}
	return stringKeys
}

func createTrie() *trieNode {
	return &trieNode{
		char:     "root",
		children: make(map[string]*trieNode),
	}
}
