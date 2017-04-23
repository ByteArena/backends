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
    console.log('Took', (duration).toFixed(2), '; mean', mean.toFixed(2));
}

function flockforces(perception) {
    const sepdist = 30; // separation distance

    let sepforce = new Vector2();   // Separation force
    let alignforce = new Vector2(); // alignment force
    let cohesionforce = new Vector2(); // cohesion force

    if(perception.External.Vision && perception.External.Vision.length) {
        const sepdistsq = sepdist * sepdist;

        for(const otheragentkey in perception.External.Vision) {
            const otheragent = perception.External.Vision[otheragentkey];
            if(otheragent.Tag === "obstacle") continue;

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
    }

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

    let followpos = new Vector2(0, perception.Specs.MaxSpeed/3);
    let aimed;

    let desired = followpos.clone();
    
    // Adding flocking behaviour (keeps agents in a cohesive pack, but separated from one another to avoid collisions)
    const flock = flockforces(perception);
    desired
    .add(flock.separation.mult(8));
    //.add(flock.alignment.mult(24))
    //.add(flock.cohesion);

    let steering = desired.clone();

    // on évite les obstacles
    let avoidanceforce = new Vector2();
    if(perception.External.Vision) {

        for(const otheragentkey in perception.External.Vision) {
            const otheragent = perception.External.Vision[otheragentkey];
            if(otheragent.Tag === "obstacle") {
                center = Vector2.fromArray(otheragent.Center);
                centerdistsq = center.magSq()
                relangle = center.angle();

                //aimed = center;

                // On passe de 0° / 360° à -180° / +180°
			    if(relangle > Math.PI) { // 180° en radians
				    relangle -= Math.PI * 2; // 360° en radian
			    }

                // On passe de 0° / 360° à -180° / +180°
                avoidanceforce.add(new Vector2(-100, 0));
            }
        }
    }


    if(avoidanceforce.x !== 0 || avoidanceforce.y !== 0) {
        steering.add(avoidanceforce).mag(perception.Specs.MaxSpeed/3)
        //aimed = steering;
    } else {
        steering.mag(perception.Specs.MaxSpeed);
    }

    // Pushing batch of mutations for this turn
    this.sendMutations(tickturn, [
        { Method: 'steer', Arguments: steering.toArray(5) },
        aimed ? (/*Math.random() < 0.95 ? null : */{ Method: 'shoot', Arguments: aimed.toArray(5) }) : null,
    ])
    /*
    .then(response => {
        measurespeed(start);
    })
    */
    .catch(err => { throw err; });

}

function RadianToDegree(rad) {
    return rad * (180.0 / Math.PI);
}

comm.connect(port, host, agentid)
.then(function({ sendRequest, sendMutations, onTick }) {
    onTick(move.bind({ sendRequest, sendMutations }));
});
