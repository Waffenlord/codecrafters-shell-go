package main

import (
	"strings"
)

type TokenType string

const (
	IDENT       = "IDENT"
	STRING      = "STRING"
	NUMBER      = "NUMBER"
	SPACE       = "SPACE"
	DOT         = "DOT"
	EOF         = "EOF"
	FORWARD     = "FORWARD"
	BACKWARD    = "BACKWARD"
	HOME        = "HOME"
	REDIRECTION = "REDIRECTION"
)

type Token struct {
	tType   TokenType
	literal string
}

type Lexer struct {
	input        string
	position     int
	readposition int
	ch           byte
}

func newLexer(i string) *Lexer {
	l := &Lexer{input: i}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readposition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readposition]
	}
	l.position = l.readposition
	l.readposition++
}

func (l *Lexer) nextToken() Token {
	var token Token

	switch l.ch {
	case '\'':
		content := l.readSingleQuote()
		token = newToken(STRING, content)
	case '"':
		content := l.readDoubleQuote()
		token = newToken(STRING, content)
	case ' ':
		token = newToken(SPACE, " ")
	case '.':
		token = newToken(DOT, ".")
	case '/':
		token = newToken(FORWARD, "/")
	case '\\':
		content := l.readBackslash()
		token = newToken(BACKWARD, content)
	case '~':
		token = newToken(HOME, "~")
	case '>':
		token = newToken(REDIRECTION, ">")
	case 0:
		token = newToken(EOF, "")
	default:
		if isLiteral(l.ch) {
			content := l.readIdentifier()
			token = newToken(IDENT, content)
			return token
		}
		if isDigit(l.ch) {
			content := l.readNumber()
			token = newToken(NUMBER, content)
			return token
		}

	}
	l.readChar()
	return token
}

func (l *Lexer) readBackslash() string {
	l.readChar()
	if l.ch != 0 {
		return string(l.input[l.position])
	}
	return ""
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLiteral(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readSingleQuote() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == 0 || l.ch == '\'' {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readDoubleQuote() string {
	selectedStrings := []string{}
	for {
		l.readChar()
		if l.ch == 0 || l.ch == '"' {
			break
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case '\\':
				selectedStrings = append(selectedStrings, "\\")
				continue
			case '"':
				selectedStrings = append(selectedStrings, "\"")
				continue
			default:
				selectedStrings = append(selectedStrings, "\\")
			}
		}
		selectedStrings = append(selectedStrings, string(l.ch))
	}
	return strings.Join(selectedStrings, "")
}

func isLiteral(ch byte) bool {
	return ('a' <= ch && ch <= 'z') ||
		('A' <= ch && ch <= 'Z') ||
		ch == '_' ||
		ch == '-'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(t TokenType, l string) Token {
	return Token{tType: t, literal: l}
}

func parseInput(i string) (string, []string) {
	parts := []Token{}
	l := newLexer(i)
	currentToken := l.nextToken()
	for currentToken.tType != EOF {
		parts = append(parts, currentToken)
		currentToken = l.nextToken()
	}

	if len(parts) == 0 {
		return "", nil
	}
	var firstSpaceIndex int
	commandLiteral := ""
	for j, t := range parts {
		if t.tType == SPACE {
			firstSpaceIndex = j
			break
		}
		commandLiteral += t.literal
	}
	cleanedParts := parts[firstSpaceIndex:]
	if len(cleanedParts) < 1 {
		return commandLiteral, nil
	}

	result := []string{}
	var previousType TokenType
	for i, t := range cleanedParts {
		switch t.tType {
		case SPACE:
			if i == 0 {
				continue
			}
			if previousType != SPACE {
				result = append(result, t.literal)
			}
			previousType = t.tType
		case REDIRECTION:
			if previousType == NUMBER {
				result[len(result)-1] = result[len(result)-1] + t.literal
				previousType = t.tType
				continue
			}
			result = append(result, t.literal)
			previousType = t.tType
		default:
			result = append(result, t.literal)
			previousType = t.tType
		}
	}

	return commandLiteral, result

}
