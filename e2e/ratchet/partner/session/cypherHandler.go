package session

type CypherHandler interface {
	AddKey(cy Cypher)
	DeleteKey(cy Cypher)
}
