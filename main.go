package main

import (
	"fmt"
	"strconv"

	"github.com/kurarrr/monkey/ast"
	"github.com/kurarrr/monkey/lexer"
	"github.com/kurarrr/monkey/token"
)

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > または <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X または !X
	CALL        // myFunction(X)
)

type Parser struct {
	l *lexer.Lexer

	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	} // 初期化

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseAddExpression)
	p.registerInfix(token.ASTERISK, p.parseMulExpression)

	// tokenを2つ進めて2つ入れる
	// null,null -> null,a[0] -> a[0],a[1]
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}
	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	// TODO: セミコロンに遭遇するまで式を読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()
	// TODO: セミコロンに遭遇するまで式を読み飛ばしてしまっている
	for !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}
func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	leftExp := prefix()
	fmt.Println("left : " + leftExp.String())
	if p.peekTokenIs(token.SEMICOLON) {
		fmt.Println("return : " + leftExp.String())
		return leftExp
	}
	// precedence 以上を parseする
	if p.peekTokenPriority() < precedence {

		return leftExp
	}
	infix := p.infixParseFns[p.peekToken.Type]
	rightExp := infix(leftExp)
	fmt.Println("right : " + rightExp.String())
	fmt.Println(leftExp.String())
	if p.peekTokenIs(token.SEMICOLON) {
		fmt.Println("return : " + leftExp.String())
		return leftExp
	}
	op := p.peekToken.Type
	pr := p.peekTokenPriority()
	// p.nextToken()
	// p.nextToken()
	return &ast.InfixExpression{
		Op:       op,
		LeftExp:  rightExp,
		RightExp: p.parseExpression(pr),
	}
}

func (p *Parser) peekTokenPriority() int {
	switch p.peekToken.Type {
	case token.PLUS:
		return SUM
	case token.ASTERISK:
		return PRODUCT
	default:
		return LOWEST
	}
}
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseAddExpression(left ast.Expression) ast.Expression {
	p.nextToken()
	p.nextToken()
	right := p.parseExpression(SUM)
	fmt.Println("cur " + string(p.curToken.Type) + "l : " + left.String() + " " + "r : " + right.String())
	return &ast.InfixExpression{
		LeftExp:  left,
		RightExp: right,
		Op:       token.PLUS,
	}
}

func (p *Parser) parseMulExpression(left ast.Expression) ast.Expression {
	p.nextToken()
	p.nextToken()
	right := p.parseExpression(PRODUCT)
	fmt.Println("cur : " + string(p.curToken.Literal) + " ,l : " + left.String() + " " + "r : " + right.String())
	return &ast.InfixExpression{
		LeftExp:  left,
		RightExp: right,
		Op:       token.ASTERISK,
	}
}

func main() {
	input := `
1*2*3;
`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	if len(program.Statements) != 0 {
		fmt.Println(program.Statements[0].String())
	}
}
