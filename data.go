package main

type Thing interface {
	Id() identifier
	Name() string
	SetId(id identifier)
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
func (t actualThing) SetId(newId identifier) {
	t.id = newId
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

type ThingAccessor struct {
	ThingGetter
	ThingSetter
}

type GetAccessorMsg struct {
	id       identifier
	response chan ThingAccessor
}

type GetAccessorByNameMsg struct {
	name     string
	response chan ThingAccessor
}

type ThingAdderMsg struct {
	thing    Thing
	response chan identifier
}

type ThingManager struct {
	getGetter         chan GetGetterMsg
	getSetter         chan GetSetterMsg
	getAccessor       chan GetAccessorMsg
	getAccessorByName chan GetAccessorByNameMsg
	add               chan ThingAdderMsg
	del               chan identifier
}

func (m ThingManager) GetThingAccessor(id identifier) ThingAccessor {
	response := make(chan ThingAccessor)
	m.getAccessor <- GetAccessorMsg{id, response}
	return <-response
}

func (m ThingManager) GetThingAccessorByName(name string) ThingAccessor {
	response := make(chan ThingAccessor)
	m.getAccessorByName <- GetAccessorByNameMsg{name, response}
	return <-response
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

func (m ThingManager) Add(t Thing) identifier {
	response := make(chan identifier)
	m.add <- ThingAdderMsg{t, response}
	return <-response
}

func (m ThingManager) Remove(id identifier) {
	m.del <- id
}

func NewThingManager() *ThingManager {
	manager := ThingManager{make(chan GetGetterMsg), make(chan GetSetterMsg), make(chan GetAccessorMsg), make(chan GetAccessorByNameMsg), make(chan ThingAdderMsg), make(chan identifier)}
	nextId := identifier(0)
	go func() {
		type thingAccessors struct {
			getter chan Thing
			setter chan SetterMsg
			closer chan bool
		}

		Things := make(map[identifier]thingAccessors)
		ThingsByName := make(map[string]thingAccessors)
		for {
			select {
			case addThing := <-manager.add:
				addThing.thing.SetId(nextId)
				nextId++
				getter := make(chan Thing)
				setter := make(chan SetterMsg)
				closer := make(chan bool)

				thingFunc := func(thing Thing, setting func(thing Thing, thingChan chan Thing)) {
					thingChan := make(chan Thing)
					for {
						select {
						case setter <- SetterMsg{thing, thingChan}:
							go setting(thing, thingChan)
							return
						case getter <- thing:
						case <-closer:
							close(getter)
							close(setter)
							return
						}
					}
				}
				var settingFunc func(thing Thing, thingChan chan Thing)
				settingFunc = func(thing Thing, thingChan chan Thing) {
					for {
						select {
						case thing = <-thingChan:
							go thingFunc(thing, settingFunc)
						case getter <- thing:
						}
					}
				}
				go thingFunc(addThing.thing, settingFunc)
				Things[addThing.thing.Id()] = thingAccessors{getter, setter, closer}
				addThing.response <- addThing.thing.Id()
			case d := <-manager.del:
				Things[d].closer <- true
				delete(Things, d)
			case g := <-manager.getAccessor:
				g.response <- ThingAccessor{Things[g.id].getter, Things[g.id].setter}
			case g := <-manager.getAccessorByName:
				g.response <- ThingAccessor{ThingsByName[g.name].getter, ThingsByName[g.name].setter}
			case g := <-manager.getGetter:
				g.response <- Things[g.id].getter
			case s := <-manager.getSetter:
				s.response <- Things[s.id].setter
			}
		}
	}()
	return &manager
}
