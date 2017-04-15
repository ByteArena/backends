const comm = require('./utils/comm');
const now = require('performance-now');
const Vector2 = require('./utils/vector2');
const { map } = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

let timecursor = 0;
const times = [];

function measurespeed(start) {
    const duration = now() - start/* - 10*/;
    times[timecursor%6000] = duration;
    timecursor++;

    const mean = times.reduce(function(carry, val) {
        return carry + val;
    }, 0) / times.length;
    //console.log('Took', (duration).toFixed(2), '; mean', mean.toFixed(2));
}

function flockforces(perception) {
    const sepdist = 30; // separation distance

    let sepforce = new Vector2();   // Separation force
    let alignforce = new Vector2(); // alignment force
    let cohesionforce = new Vector2(); // cohesion force

    const sepdistsq = sepdist * sepdist;

    for(const otheragentkey in perception.External.Vision) {
        const otheragent = perception.External.Vision[otheragentkey];
        const othervelocity = Vector2.fromArray(otheragent.Velocity);
        const otherposition = Vector2
            .fromArray(otheragent.Center)
            .add(othervelocity);
        
        otherposition.mag(otherposition.mag() - otheragent.Radius)

        if(otherposition.magSq() <= sepdistsq) {
            sepforce.sub(otherposition);
        }

        cohesionforce.add(otherposition);
        alignforce.add(othervelocity);
    }

    alignforce.div(perception.External.Vision.length);
    cohesionforce.div(perception.External.Vision.length);

    return {
        separation: sepforce,
        alignment: alignforce,
        cohesion: cohesionforce,
    };
}

function findAttractor(perception) {
      // Finding the attractor
    let attractor = null;
    for(const otheragentkey in perception.External.Vision) {
        const otheragent = perception.External.Vision[otheragentkey];
        if(otheragent.Tag === "attractor") {
            attractor = otheragent;
            break;
        }
    }

    if(attractor === null) return null;

    return {
        position: Vector2.fromArray(attractor.Center),
        velocity: Vector2.fromArray(attractor.Velocity),
    }
}

function move(tickturn, perception) {

    const start = now();

    const attractor = findAttractor(perception);
    if(attractor === null) return;

    const curvelocity = Vector2.fromArray(perception.Internal.Velocity);

    //
    // Following some pixels behind the attractor
    //
    const followpos = attractor.position
        .clone()
        .add(attractor.velocity)                    // attractor next position
        .sub(attractor.velocity.clone().mag(60))    // 60px behind him
    
    // Determine adapted speed with regards to distance with target
    const disttotarget = followpos.mag();
    let speed = perception.Specs.MaxSpeed;
    if(disttotarget < 60) {
        speed = map(disttotarget, 0, perception.Specs.MaxSpeed, 0, attractor.velocity.mag());
    }
    
    // Adding flocking behaviour (keeps agents in a cohesive pack, but separated from one another to avoid collisions)
    const flock = flockforces(perception);

    // Determine steering based on following point, and flock forces
    desired = followpos.clone()
        .add(flock.separation.mult(32))
        .add(flock.alignment.mult(24))
        .add(flock.cohesion);

    const steering = desired.clone();//.limit(0.1);

    //
    // Shooting straight at next attractor position
    //
    const aimed = attractor.position.clone().add(attractor.velocity);

    // Pushing batch of mutations for this turn
    this.sendMutations(tickturn, [
        { Method: 'steer', Arguments: steering.toArray(5) }, // 3: précision
        Math.random() < 0.95 ? null : { Method: 'shoot', Arguments: aimed.toArray(5) }, // 3: précision
    ])
    .then(response => {
        measurespeed(start);
    })
    .catch(err => { throw err; });

}

comm.connect(port, host, agentid)
.then(function({ sendRequest, sendMutations, onTick }) {
    onTick(move.bind({ sendRequest, sendMutations }));
});
