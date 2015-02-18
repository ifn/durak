package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	sm "github.com/ifn/go-statemachine"
)

type DeskMsg struct {
	Desk []string `json:"desk"`
}

type PlayerMsg struct {
	Cmd  sm.EventType `json:"command"`
	Card string       `json:"card"`
}

//

var Suits []string = []string{
	"S",
	"C",
	"H",
	"D",
}

var CardValues map[string]int = map[string]int{
	"6": 6, "7": 7, "8": 8, "9": 9, "10": 10,
	"J": 11,
	"Q": 12,
	"K": 13,
	"A": 14,
}

func higher(c0, c1, t string) int {
	// c0 and c1 have the same suit
	if c0[0] == c1[0] {
		if CardValues[c0[1:]] > CardValues[c1[1:]] {
			return 1
		}
		if CardValues[c0[1:]] < CardValues[c1[1:]] {
			return -1
		}
		return 0
	}
	// c0 is trump, c1 is not
	if c0[:1] == t {
		return 1
	}
	// c1 is trump, c0 is not
	if c1[:1] == t {
		return -1
	}
	// suits are different, both are not trump
	return -2
}

var CardRE *regexp.Regexp = regexp.MustCompile(`[SCHD]([6-9JQKA]|10)`)

func isValid(c string) bool {
	return CardRE.MatchString(c)
}

//

const (
	cmdStart sm.EventType = iota
	cmdMove

	cmdCount
)

const (
	stateCollection sm.State = iota

	stateAttack
	stateDefense

	stateCount
)

type roundResult int

const (
	None roundResult = iota
	Beat
	NotBeat
)

func stateToString(s sm.State) string {
	return [...]string{
		stateCollection: "COLLECTION",
		stateDefense:    "DEFENSE",
		stateAttack:     "ATTACK",
	}[s]
}

func cmdToString(t sm.EventType) string {
	return [...]string{
		cmdStart: "START",
		cmdMove:  "MOVE",
	}[t]
}

//

type cmdArgs struct {
	conn *playerConn
	card string
}

//

func logOutOfTurn(pconn *playerConn) {
	log.Printf("out of turn: %v", pconn.conn.RemoteAddr())
}

func logWontBeat(c1, c2, t string) {
	log.Printf("%v won't bit %v, trump is %v", c1, c2, t)
}

func logWrongNumber(n int) {
	log.Printf("wrong number of players: %v", n)
}

func logStrangeCard(c string, pc *playerConn) {
	log.Printf("%v doesn't own %v", pc, c)
}

//

type gameState struct {
	// 1. fields that don't change during a game

	sm  *sm.StateMachine
	hub *hub

	// trump suit
	trump string

	// 2. fields that don't change during a round

	deck []string

	// attacker that started a round
	aconnStart *playerConn
	// defender
	dconn *playerConn

	// 3. fields that change during a round

	desk []string

	// attacker
	aconn *playerConn
	// card that should be beaten
	cardToBeat string
}

func (self *gameState) popCard() (card string) {
	if deck := self.deck; len(deck) > 0 {
		card = deck[0]
		self.deck = deck[1:]
	}
	return
}

func (self *gameState) initDeck() {
	numCards := len(Suits) * len(CardValues)

	deck := make([]string, 0, numCards)
	for _, suit := range Suits {
		for cv := range CardValues {
			deck = append(deck, suit+cv)
		}
	}

	order := rand.Perm(numCards)
	for _, pos := range order {
		self.deck = append(self.deck, deck[pos])
	}
}

//TODO: don't like it
func (self *gameState) setTrump(card string) {
	tcard := self.popCard()
	if tcard == "" {
		tcard = card
	} else {
		self.deck = append(self.deck, tcard)
	}
	self.trump = tcard[:1]
}

func (self *gameState) nextPlayer(c *playerConn) *playerConn {
	return self.hub.conns.(*mapRing).Next(c).(*playerConn)
}

func (self *gameState) chooseStarting() *playerConn {
	conns := self.hub.conns.(*mapRing)

	return conns.Nth(rand.Intn(conns.Len())).(*playerConn)
}

func (self *gameState) markInactive() {
	if len(self.deck) == 0 {
		for pc := range self.hub.conns.Enumerate() {
			if pc := pc.(*playerConn); len(pc.cards) == 0 {
				pc.active = false
			}
		}
	}
}

func (self *gameState) firstActive(c *playerConn) *playerConn {
	for pc := range self.hub.conns.(*mapRing).EnumerateFrom(c) {
		if pc := pc.(*playerConn); pc.active {
			return pc
		}
	}
	return nil
}

func (self *gameState) setRoles(res roundResult) {
	switch res {
	case None:
		self.aconn = self.chooseStarting()
	case Beat:
		self.aconn = self.dconn
	case NotBeat:
		self.aconn = self.nextPlayer(self.dconn)
	}
	self.dconn = self.nextPlayer(self.aconn)
	self.aconnStart = self.aconn
}

func (self *gameState) dealCards() (card string) {
	for i := 0; i < 6; i++ {
		for pc := range self.hub.conns.Enumerate() {
			card = pc.(*playerConn).fromDeck()
		}
	}
	return
}

func (self *gameState) takeCards() {
	conns := self.hub.conns.(*mapRing)

	takeCards := func(pc *playerConn) {
		for len(self.deck) > 0 && len(pc.cards) < 6 {
			pc.fromDeck()
		}
	}

	for pc := range conns.EnumerateFrom(self.aconnStart) {
		if pc := pc.(*playerConn); pc != self.dconn {
			takeCards(pc)
		}
	}
	takeCards(self.dconn)
}

func (self *gameState) newRound(res roundResult) {
	switch res {
	case None:
		self.initDeck()
		card := self.dealCards()
		self.setTrump(card)
	case NotBeat:
		self.dconn.fromDesk()
		self.takeCards()
	case Beat:
		self.desk = self.desk[:0]
		self.takeCards()
	}
	self.markInactive()
	self.setRoles(res)
	self.cardToBeat = ""
}

// event handlers
// event handlers are actually transition functions.
// in case error event handler should neither change the gameState,
// nor return the state value different from passed to it as an argument.

func (self *gameState) handleStartInCollection(s sm.State, e *sm.Event) sm.State {
	if n := self.hub.conns.(*mapRing).Len(); n < 2 || n > 6 {
		logWrongNumber(n)
		return s
	}

	self.newRound(None)
	return stateAttack
}

func (self *gameState) handleMoveInAttack(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	// check that it's conn's turn to move
	if conn != self.aconn {
		logOutOfTurn(conn)
		return s
	}

	// attacker sent the card
	if card != "" {
		// throw card to desk
		if _, ok := conn.cards[card]; !ok {
			logStrangeCard(card, conn)
			return s
		}
		conn.toDesk(card)

		self.cardToBeat = card
		return stateDefense
	}

	// attacker sent no card

	aconn := self.nextPlayer(self.aconn)
	if aconn == self.dconn {
		aconn = self.nextPlayer(aconn)
	}

	// check if all attackers have been polled
	if aconn == self.aconnStart {
		self.newRound(Beat)
		return stateAttack
	}

	self.aconn = aconn
	return stateAttack
}

func (self *gameState) handleMoveInDefense(s sm.State, e *sm.Event) sm.State {
	conn := e.Data.(cmdArgs).conn
	card := e.Data.(cmdArgs).card

	// check that it's conn's turn to move
	if conn != self.dconn {
		logOutOfTurn(conn)
		return s
	}

	// defender takes the cards
	if card == "" {
		self.newRound(NotBeat)
		return stateAttack
	}

	// check that the sent card is capable to beat
	if higher(card, self.cardToBeat, self.trump) != 1 {
		logWontBeat(card, self.cardToBeat, self.trump)
		return s
	}

	// throw card to desk
	if _, ok := conn.cards[card]; !ok {
		logStrangeCard(card, conn)
		return s
	}
	conn.toDesk(card)

	return stateAttack
}

func (self *gameState) showDesk(s sm.State, e *sm.Event) sm.State {
	desk, err := json.Marshal(DeskMsg{self.desk})
	if err != nil {
		log.Println(err)
		return s
	}

	self.hub.bcastChan <- desk

	return s
}

func (self *gameState) log(s sm.State, e *sm.Event) sm.State {
	log.Println(self)
	return s
}

//

func NewGameState() *gameState {
	gst := new(gameState)

	gst.sm = sm.New(stateCollection, uint(stateCount), uint(cmdCount))

	gst.sm.OnChain(cmdStart,
		[]sm.State{stateCollection},
		[]sm.EventHandler{
			gst.handleStartInCollection,
			gst.showDesk,
			gst.log,
		},
	)

	gst.sm.On(cmdMove,
		[]sm.State{stateAttack},
		gst.handleMoveInAttack,
	)
	gst.sm.On(cmdMove,
		[]sm.State{stateDefense},
		gst.handleMoveInDefense,
	)

	gst.sm.OnChain(cmdMove,
		[]sm.State{stateAttack, stateDefense},
		[]sm.EventHandler{
			gst.showDesk,
			gst.log,
		},
	)

	gst.deck = make([]string, 0, len(Suits)*len(CardValues)+ /*for trump*/ 1)
	gst.desk = make([]string, 0, 12)

	gst.hub = NewHub()

	return gst
}

//

type playerConn struct {
	gst *gameState

	cards  map[string]struct{}
	active bool

	conn      *websocket.Conn
	hubToConn chan []byte
}

func (self *playerConn) fromDeck() (card string) {
	if card = self.gst.popCard(); card != "" {
		self.cards[card] = struct{}{}
	}
	return
}

func (self *playerConn) toDesk(card string) {
	delete(self.cards, card)

	self.gst.desk = append(self.gst.desk, card)
}

func (self *playerConn) fromDesk() {
	for _, card := range self.gst.desk {
		self.cards[card] = struct{}{}
	}

	self.gst.desk = self.gst.desk[:0]
}

func (self *playerConn) write() {
	defer func() {
		err := self.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	for {
		select {
		case m, ok := <-self.hubToConn:
			if !ok {
				return
			}
			//TODO: text or binary?
			err := self.conn.WriteMessage(websocket.TextMessage, m)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func (self *playerConn) read() {
	defer func() {
		err := self.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	for {
		var m PlayerMsg

		err := self.conn.ReadJSON(&m)
		if err != nil {
			log.Println(err)
			return
		}

		var event *sm.Event
		switch m.Cmd {
		case cmdStart:
			event = &sm.Event{cmdStart, nil}
		case cmdMove:
			event = &sm.Event{cmdMove, cmdArgs{self, m.Card}}
		default:
			log.Printf("unknown command: %v", m.Cmd)
			continue
		}

		err = self.gst.sm.Emit(event)
		if err != nil {
			log.Println(err)
		}
	}
}

//

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func playerHandler(gst *gameState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		p := &playerConn{gst, make(map[string]struct{}), true, conn, make(chan []byte)}

		gst.hub.regChan <- p
		defer func() {
			gst.hub.unregChan <- p
		}()

		go p.write()
		p.read()
	}
}

//

func startDurakSrv() error {
	gst := NewGameState()

	http.HandleFunc("/", playerHandler(gst))

	return http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func main() {
	err := startDurakSrv()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())

	runtime.GOMAXPROCS(runtime.NumCPU())
}
