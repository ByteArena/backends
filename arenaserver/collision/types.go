package collision

import (
	"strconv"
	"sync"

	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
)

/////////////////////////////////////////////////////////////////////////////////////////

type FinelyCollisionable interface {
	GetPointA() vector.Vector2
	GetPointB() vector.Vector2
	GetRadius() float64
	GetType() int
	GetID() string
}

/////////////////////////////////////////////////////////////////////////////////////////

type Collision struct {
	ColliderType      int
	ColliderID        string
	CollideeType      int
	CollideeID        string
	Point             vector.Vector2
	ColliderTimeBegin float64 // from 0 to 1, 0 = beginning of tick, 1 = end of tick
	ColliderTimeEnd   float64
	CollideeTimeBegin float64 // from 0 to 1, 0 = beginning of tick, 1 = end of tick
	CollideeTimeEnd   float64
	ColliderMovement  *MovementState
}

type CollisionByTimeAsc []Collision

func (a CollisionByTimeAsc) Len() int      { return len(a) }
func (a CollisionByTimeAsc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a CollisionByTimeAsc) Less(i, j int) bool {
	return a[i].ColliderTimeBegin < a[j].ColliderTimeBegin
}

////////////////////////////////////////////////////////////////////////////////////////

type MovementState struct {
	Type   int
	ID     string
	Before CollisionMovingObjectState
	After  CollisionMovingObjectState
	Rect   *rtreego.Rect
}

func (geobj MovementState) Bounds() *rtreego.Rect {
	return geobj.Rect
}

func (geobj *MovementState) GetPointA() vector.Vector2 {
	return geobj.Before.Position
}

func (geobj *MovementState) GetPointB() vector.Vector2 {
	return geobj.After.Position
}

func (geobj *MovementState) GetRadius() float64 {
	return geobj.After.Radius
}

func (geobj *MovementState) GetType() int {
	return geobj.Type
}

func (geobj *MovementState) GetID() string {
	return geobj.ID
}

func MovementStateComparator(obj1, obj2 rtreego.Spatial) bool {
	sp1 := obj1.(*MovementState)
	sp2 := obj2.(*MovementState)

	return sp1.Type == sp2.Type && sp1.ID == sp2.ID
}

////////////////////////////////////////////////////////////////////
type CollisionMovingObjectState struct {
	Position vector.Vector2
	Velocity vector.Vector2
	Radius   float64
}

//////////////////////////////////////////////////////////////////////

// Memoization pas parfaite car la position du collider au moment de la collision est différente de celle du collidee
// Pour le rendre possible, il faudrait consigner la position du point de collision plutôt que celle de l'objet au moment de la collision
// Et résoudre la position de l'objet avec la collision en tangente du cercle de surface de l'objet sur sa trajectoire par après
// Utilisé néanmoins pour mieux faire correspondre visuellement la position de collision détectée pour chaque couple collider/collidee
// Empêche l'utilisation de goroutines pour traiter en parallèle les collisions moving/moving
type memoizedMovingMovingCollisions struct {
	collisions map[string]*Collision
	mutex      *sync.RWMutex
}

func NewMemoizedMovingMovingCollisions() *memoizedMovingMovingCollisions {
	return &memoizedMovingMovingCollisions{
		collisions: make(map[string]*Collision),
		mutex:      &sync.RWMutex{},
	}
}

func (m *memoizedMovingMovingCollisions) add(colls []Collision) {
	m.mutex.Lock()
	for _, coll := range colls {
		m.collisions[strconv.Itoa(coll.CollideeType)+":"+coll.CollideeID+","+strconv.Itoa(coll.ColliderType)+":"+coll.ColliderID] = &Collision{
			ColliderType:      coll.CollideeType,
			ColliderID:        coll.CollideeID,
			CollideeType:      coll.ColliderType,
			CollideeID:        coll.ColliderID,
			Point:             coll.Point,
			ColliderTimeBegin: coll.CollideeTimeBegin,
			ColliderTimeEnd:   coll.CollideeTimeEnd,
			CollideeTimeBegin: coll.ColliderTimeBegin,
			CollideeTimeEnd:   coll.ColliderTimeEnd,
		}
	}
	m.mutex.Unlock()
}

func (m *memoizedMovingMovingCollisions) get(colliderType int, colliderID string, collideeType int, collideeID string) *Collision {
	var res *Collision

	m.mutex.RLock()
	res, ok := m.collisions[strconv.Itoa(colliderType)+":"+colliderID+","+strconv.Itoa(collideeType)+":"+collideeID]
	m.mutex.RUnlock()

	if !ok {
		return nil
	}

	return res
}

/////////////////////////////////////////////////////////////////////

type collisionHandlerFunc func(collision vector.Vector2, geoObject FinelyCollisionable)

/////////////////////////////////////////////////////////////////////

type collisionWrapper struct {
	Point    vector.Vector2
	Obstacle FinelyCollisionable
}
