package main

type thing interface {
	Id() identifier
	Name() string
	Brief() string
	Long() string
}

type actualThing struct {
	id    identifier
	name  string
	brief string
	long  string
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

func getThingGetter(id identifier, getGetter chan struct {
	id       identifier
	response chan chan thing
},) chan thing {
	response := make(chan chan thing)
	getGetter <- struct {
		id       identifier
		response chan chan thing
	}{id, response}
	return <-response
}

func getThing(id identifier, getGetter chan struct {
	id       identifier
	response chan chan thing
},) thing {
	getter := getThingGetter(id, getGetter)
	return <-getter
}

func getThingSetter(id identifier, getSetter chan struct {
	id       identifier
	response chan chan struct {
		it  thing
		set chan thing
	}
},) chan struct {
	it  thing
	set chan thing
} {
	response := make(chan chan struct {
		it  thing
		set chan thing
	})
	getSetter <- struct {
		id       identifier
		response chan chan struct {
			it  thing
			set chan thing
		}
	}{id, response}
	return <-response
}

//func addThing

func initThingManager() (
	chan struct {
		id       identifier
		response chan chan thing
	},
	chan struct {
		id       identifier
		response chan chan struct {
			it  thing
		        set chan thing
		}
	},
	chan thing,
	chan identifier) {

	getGetter := make(chan struct {
		id       identifier
		response chan chan thing
	})
	getSetter := make(chan struct {
		id       identifier
		response chan chan struct {
			it  thing
			set chan thing
		}
	})
	add := make(chan thing)
	del := make(chan identifier)
	go func() {
		things := make(map[identifier]struct {
			getter chan thing
			setter chan struct {
				it  thing
				set chan thing
			}
			closer chan bool
		})
		for {
			select {
			case a := <-add:
				getter := make(chan thing)
				setter := make(chan struct {
					it  thing
					set chan thing
				})
				closer := make(chan bool)
				getter <- a
				go func() {
					it := <-getter
					itc := make(chan thing)
					for {
						select {
						case setter <- struct {
							it  thing
							set chan thing
						}{it: it, set: itc}:
							it = <-itc
						case getter <- it:
						case <-closer:
							close(getter)
							close(setter)
							return
						}
					}
				}()
				things[a.Id()] = struct {
					getter chan thing
					setter chan struct {
						it  thing
						set chan thing
					}
					closer chan bool
				}{getter, setter, closer}
			case d := <-del:
				things[d].closer <- true
				delete(things, d)
			case g := <-getGetter:
				g.response <- things[g.id].getter
			case s := <-getSetter:
				s.response <- things[s.id].setter
			}
		}
	}()
	return getGetter, getSetter, add, del
}
