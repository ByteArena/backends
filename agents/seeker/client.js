const comm = require('./utils/comm');
const Vector2 = require('./utils/vector2');
const { map } = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

function move(tickturn, perception) {

    let followpos = new Vector2(0, perception.Specs.MaxSpeed/3);
    let desired = followpos.clone();
    let steering = desired.clone();

    // on évite les obstacles
    let avoidanceforce = new Vector2();
    if(perception.External.Vision) {

        for(const otheragentkey in perception.External.Vision) {
            const otheragent = perception.External.Vision[otheragentkey];
            if(otheragent.Tag !== "obstacle") continue;

            closeEdge = Vector2.fromArray(otheragent.CloseEdge);
            farEdge = Vector2.fromArray(otheragent.FarEdge);

            //console.log('close', closeEdge, 'far', farEdge);

            //center = Vector2.fromArray(otheragent.Center);
            avoidanceforce.sub(closeEdge.clone().mag(100));
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
    ])
    .catch(err => { throw err; });

}

function RadianToDegree(rad) {
    return rad * (180.0 / Math.PI);
}

comm.connect(port, host, agentid)
.then(function({ sendRequest, sendMutations, onTick }) {
    onTick(move.bind({ sendRequest, sendMutations }));
});
