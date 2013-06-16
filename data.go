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

type GetGetterMsg struct {
	id       identifier
	response chan chan Thing
}

func getThingGetter(id identifier, getGetter chan GetGetterMsg) chan Thing {
	response := make(chan chan Thing)
	getGetter <- GetGetterMsg{id, response}
	return <-response
}

func getThing(id identifier, getGetter chan GetGetterMsg) Thing {
	getter := getThingGetter(id, getGetter)
	return <-getter
}

type SetterMsg struct {
	it  Thing
	set chan Thing
}

type GetSetterMsg struct {
	id       identifier
	response chan chan SetterMsg
}

func getThingSetter(id identifier, getSetter chan GetSetterMsg) chan SetterMsg {
	response := make(chan chan SetterMsg)
	getSetter <- GetSetterMsg{id, response}
	return <-response
}

//func addThing

func initThingManager() (
	chan GetGetterMsg,
	chan GetSetterMsg,
	chan Thing,
	chan identifier) {

	getGetter := make(chan GetGetterMsg)
	getSetter := make(chan GetSetterMsg)
	add := make(chan Thing)
	del := make(chan identifier)
	go func() {
		Things := make(map[identifier]struct {
			getter chan Thing
			setter chan SetterMsg
			closer chan bool
		})
		for {
			select {
			case a := <-add:
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
				Things[a.Id()] = struct {
					getter chan Thing
					setter chan SetterMsg
					closer chan bool
				}{getter, setter, closer}
			case d := <-del:
				Things[d].closer <- true
				delete(Things, d)
			case g := <-getGetter:
				g.response <- Things[g.id].getter
			case s := <-getSetter:
				s.response <- Things[s.id].setter
			}
		}
	}()
	return getGetter, getSetter, add, del
}
