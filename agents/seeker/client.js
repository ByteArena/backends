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

    //
    // Implementing Reynolds' steering behavior
    // http://www.red3d.com/cwr/steer/
    //

    // determine the steering force to apply to reach the attractor
    const attractorpos = Vector2.fromArray(perception.Objective.Attractor);
    const curvelocity = Vector2.fromArray(perception.Internal.Velocity);

    const disttotarget = attractorpos.mag();
    let speed = perception.Specs.MaxSpeed;
    if(disttotarget < perception.Internal.Proprioception) {
        // arrival, slow down
        speed = map(disttotarget, 0, perception.Internal.Proprioception, 0, perception.Specs.MaxSpeed);
    }

    const desired = attractorpos
        .clone()
        .mag(speed);

    const steering = desired
        .clone()
        .sub(curvelocity)
        .limit(perception.Specs.MaxSteeringForce);
        

    // Pushing batch of mutations for this turn
    this.sendMutations(tickturn, [
        { Method: 'steer', Arguments: steering.toArray(5) }, // 3: précision
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
