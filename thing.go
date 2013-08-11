package main

import (
	"fmt"
)

/// @todo change channels to be unidirectional

type Thing interface {
	Id() identifier
	Name() string
	SetId(id identifier)
}

type actualThing struct {
	id   identifier
	name string
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

type ThingGetter chan Thing

func (g ThingGetter) Get() (Thing, bool) {
	thing, ok := <-g
	return thing, ok
}

type SetterMsg struct {
	it        Thing
	set       chan Thing
	chainTime chan ChainTime
}

type ThingSetter chan SetterMsg

func (s ThingSetter) Set(modify func(t *Thing)) (ok bool) {
	msg, ok := <-s
	if !ok {
		return false
	}
	msg.chainTime <- NotChaining
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

type ThingSetTime chan ChainTime

type ThingAccessor struct {
	ThingGetter
	ThingSetter
	ThingSetTime
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

type ThingSaver struct {
	add    chan Thing
	del    chan identifier
	change chan Thing
}

type ThingManager struct {
	getGetter         chan GetGetterMsg
	getSetter         chan GetSetterMsg
	getAccessor       chan GetAccessorMsg
	getAccessorByName chan GetAccessorByNameMsg
	add               chan ThingAdderMsg
	dbAdd             chan Thing
	del               chan identifier
	saver             ThingSaver
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

func (m ThingManager) DbAdd(t Thing) {
	m.dbAdd <- t
}

func (m ThingManager) Remove(id identifier) {
	m.del <- id
}

func (a ThingAccessor) TryGet(chainTime ChainTime) (setter SetterMsg, ok bool, reset bool) {
	if a.ThingSetter == nil || a.ThingGetter == nil {
		fmt.Println("accessor setter nil")
		return SetterMsg{}, false, false
	}
	select {
	case setter, ok := <-a.ThingSetter:
		setter.chainTime <- chainTime
		fmt.Println("returning setter response")
		return setter, ok, false
	case time, ok := <-a.ThingSetTime:
		if !ok {
			fmt.Println("TryGet error: ThingSetTime nil.")
			return SetterMsg{}, true, false
		}
		if time < chainTime {
			//			fmt.Println("Tryget chaintime preempted " + chainTime.String() + " by " + time.String())
			return SetterMsg{}, true, true
		}
		setter, ok := <-a.ThingSetter
		setter.chainTime <- chainTime
		fmt.Println("Tryget returning set success")
		return setter, ok, false
	}
	fmt.Println("Tryget got where it shouldn't")
	return SetterMsg{}, false, false // this should never get hit
}

func (m ThingManager) GetById(id identifier) (Thing, bool) {
	accessor := ThingManager(m).GetThingAccessor(id)
	if accessor.ThingGetter == nil {
		fmt.Println("ThingManager.GetById error: ThingGetter nil " + id.String())
		return nil, false
	}
	thing, ok := <-accessor.ThingGetter
	return thing, ok
}

func ReleaseThings(things []SetterMsg) {
	for _, a := range things {
		a.set <- a.it
	}
}

func NewThingManager() *ThingManager {
	manager := ThingManager{
		getGetter:         make(chan GetGetterMsg),
		getSetter:         make(chan GetSetterMsg),
		getAccessor:       make(chan GetAccessorMsg),
		getAccessorByName: make(chan GetAccessorByNameMsg),
		add:               make(chan ThingAdderMsg),
		dbAdd:             make(chan Thing),
		del:               make(chan identifier),
		saver: ThingSaver{
			add:    make(chan Thing, 1000),
			del:    make(chan identifier, 1000),
			change: make(chan Thing, 1000),
		},
	}
	go func() {
		type thingAccessors struct {
			getter        chan Thing
			setter        chan SetterMsg
			closer        chan bool
			setTimeGetter chan ChainTime
		}

		Things := make(map[identifier]thingAccessors)
		ThingsByName := make(map[string]thingAccessors)
		ThingNameMap := make(map[identifier]string)

		doAdd := func(thing Thing) {
			getter := make(chan Thing)
			setter := make(chan SetterMsg)
			closer := make(chan bool)
			setTimeGetter := make(chan ChainTime)
			thingFunc := func(thing Thing, setting func(thing Thing, thingChan chan Thing, time ChainTime)) {
				thingChan := make(chan Thing)
				timeSetter := make(chan ChainTime)
				for {
					select {
					case setter <- SetterMsg{thing, thingChan, timeSetter}:
//						fmt.Println("locking " + thing.Id().String())
						time := <-timeSetter
//						fmt.Println("locked " + thing.Id().String())
						go setting(thing, thingChan, time)
						return
					case getter <- thing:
					case setTimeGetter <- 0:
					case <-closer:
						close(getter)
						close(setter)
						return

					}
				}
			}
			var settingFunc func(thing Thing, thingChan chan Thing, time ChainTime)
			settingFunc = func(thing Thing, thingChan chan Thing, time ChainTime) {
				for {
					select {
					case t := <-thingChan:
						go thingFunc(t, settingFunc)
						manager.saver.change <- t
						it, ok := t.(*Item)
						if ok {
							fmt.Println("manager saver.change " + it.Id().String() + " at " + it.Location.String())
						}
//						fmt.Println("unlocked " + thing.Id().String())
						return
					case getter <- thing:
					case setTimeGetter <- time:
					}
				}
			}
			go thingFunc(thing, settingFunc)
			Things[thing.Id()] = thingAccessors{getter, setter, closer, setTimeGetter}
			ThingsByName[thing.Name()] = thingAccessors{getter, setter, closer, setTimeGetter}
			ThingNameMap[thing.Id()] = thing.Name()
		}

		for {
			select {
			case addThing := <-manager.add:
				addThing.thing.SetId(<-NextId)
				doAdd(addThing.thing)
				addThing.response <- addThing.thing.Id()
				fmt.Println("debug adding saver thing")
				manager.saver.add <- addThing.thing
				fmt.Println("debug added saver thing")
			case thing := <-manager.dbAdd:
				doAdd(thing)
			case d := <-manager.del:
				Things[d].closer <- true
				delete(ThingsByName, ThingNameMap[d])
				delete(Things, d)
				delete(ThingNameMap, d)
				manager.saver.del <- d
			case g := <-manager.getAccessor:
				g.response <- ThingAccessor{Things[g.id].getter, Things[g.id].setter, Things[g.id].setTimeGetter}
			case g := <-manager.getAccessorByName:
				g.response <- ThingAccessor{ThingsByName[g.name].getter, ThingsByName[g.name].setter, ThingsByName[g.name].setTimeGetter}
			case g := <-manager.getGetter:
				g.response <- Things[g.id].getter
			case s := <-manager.getSetter:
				s.response <- Things[s.id].setter
			}
		}
	}()
	return &manager
}
