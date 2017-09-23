package game

type GameEventSubscription int32

type GameInterface interface {
	ImplementsGameInterface()
	Subscribe(event string, cbk func(data interface{})) GameEventSubscription
	Unsubscribe(subscription GameEventSubscription)
	Step(dt float64)
}
