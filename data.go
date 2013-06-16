package main

type Thing interface {
	Id() identifier
	Name() string
	Brief() string
	Long() string
	Location() identifier
	SetLocation(l identifier)
	Presences() map[identifier]bool
	AddPresence(p identifier)
	RemovePresence(p identifier)
}

type actualThing struct {
	id        identifier
	name      string
	brief     string
	long      string
	location  identifier
	presences map[identifier]bool
}

func (t actualThing) Id() identifier {
	return t.id
}
func (t actualThing) Name() string {
	return t.name
}
func (t actualThing) Brief() string {
	return t.brief
}
func (t actualThing) Long() string {
	return t.long
}
func (t actualThing) Location() identifier {
	return t.location
}
func (t actualThing) SetLocation(l identifier) {
	t.location = l
}
func (t actualThing) Presences() map[identifier]bool {
	return t.presences
}
func (t actualThing) AddPresence(p identifier) {
	t.presences[p] = true
}
func (t actualThing) RemovePresence(p identifier) {
	delete(t.presences, p)
}

type ThingGetter chan Thing

func (g ThingGetter) Get() (Thing, bool) {
	thing, ok := <-g
	return thing, ok
}

type SetterMsg struct {
	it  Thing
	set chan Thing
}

type ThingSetter chan SetterMsg

func (s ThingSetter) Set(modify func(t *Thing)) (ok bool) {
	msg, ok := <-s
	if !ok {
		return false
	}
	modify(&msg.it)
	msg.set <- msg.it
	return true
}

type GetGetterMsg struct {
	id       identifier
	response chan ThingGetter
}

type GetSetterMsg struct {
	id       identifier
	response chan ThingSetter
}

type ThingManager struct {
	getGetter chan GetGetterMsg
	getSetter chan GetSetterMsg
	add       chan Thing
	del       chan identifier
}

func (m ThingManager) GetThingGetter(id identifier) ThingGetter {
	response := make(chan ThingGetter)
	m.getGetter <- GetGetterMsg{id, response}
	return <-response
}

func (m ThingManager) GetThingSetter(id identifier) ThingSetter {
	response := make(chan ThingSetter)
	m.getSetter <- GetSetterMsg{id, response}
	return <-response
}

func (m ThingManager) Add(t Thing) {
	m.add <- t
}

func (m ThingManager) Remove(id identifier) {
	m.del <- id
}

func initThingManager() ThingManager {
	manager := ThingManager{make(chan GetGetterMsg), make(chan GetSetterMsg), make(chan Thing), make(chan identifier)}
	go func() {
		type thingAccessors struct {
			getter chan Thing
			setter chan SetterMsg
			closer chan bool
		}

		Things := make(map[identifier]thingAccessors)
		for {
			select {
			case a := <-manager.add:
				getter := make(chan Thing)
				setter := make(chan SetterMsg)
				closer := make(chan bool)
				getter <- a

				go func() {
					it := <-getter
					itc := make(chan Thing)
					for {
						select {
						case setter <- SetterMsg{it, itc}:
							it = <-itc
						case getter <- it:
						case <-closer:
							close(getter)
							close(setter)
							return
						}
					}
				}()

				Things[a.Id()] = thingAccessors{getter, setter, closer}
			case d := <-manager.del:
				Things[d].closer <- true
				delete(Things, d)
			case g := <-manager.getGetter:
				g.response <- Things[g.id].getter
			case s := <-manager.getSetter:
				s.response <- Things[s.id].setter
			}
		}
	}()
	return manager
}
