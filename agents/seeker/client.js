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

function move(tickturn, perception) {

    const start = now();

      // Finding the attractor
    let attractor = null;
    for(const otheragentkey in perception.External.Vision) {
        const otheragent = perception.External.Vision[otheragentkey];
        if(otheragent.Tag === "attractor") {
            attractor = otheragent;
            break;
        }
    }

    if(attractor === null) return;
    const attractorpos = Vector2.fromArray(attractor.Center);
    const attractorvelocity = Vector2.fromArray(attractor.Velocity);
    
    followpos = attractorpos
        .clone()
        .add(attractorvelocity)
        .sub(attractorvelocity.clone().mag(30))
        .sub(attractorpos.clone().mag(Math.random() * 50 + 100));

    //
    // Implementing Reynolds' steering behavior
    // http://www.red3d.com/cwr/steer/
    //

    // determine the steering force to apply to reach the attractor
    const curvelocity = Vector2.fromArray(perception.Internal.Velocity);
    
    // Flocking behaviour    
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

        if(otherposition.magSq() <= sepdistsq) {
            sepforce.sub(otherposition);
        }

        cohesionforce.add(otherposition);
        alignforce.add(othervelocity);
    }

    alignforce.div(perception.External.Vision.length);
    cohesionforce.div(perception.External.Vision.length);

    desired = followpos.clone()
        .add(sepforce.mult(16))
        .add(alignforce.mult(24))
        .add(cohesionforce);
    
    const disttotarget = followpos.mag();
    let speed = perception.Specs.MaxSpeed;
    if(disttotarget < 30) {
        speed = map(disttotarget, 0, perception.Specs.MaxSpeed, 0, attractorvelocity.mag());
    }

    const steering = desired
        .clone()
        .limit(speed)
        .sub(curvelocity);
    
    const aimed = attractorpos.clone().add(attractorvelocity);
        

    // Pushing batch of mutations for this turn
    this.sendMutations(tickturn, [
        { Method: 'steer', Arguments: steering.toArray(5) }, // 3: précision
        Math.random() < 0.9 ? null : { Method: 'shoot', Arguments: aimed.toArray(5) }, // 3: précision
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
