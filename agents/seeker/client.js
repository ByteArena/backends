const comm = require('./utils/comm');
const Vector2 = require('./utils/vector2');
const { map } = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

// taked from http://stackoverflow.com/a/37865332/3528924
function pointInRectangle(vP, vRA, vRB, vRC, vRD) {
    //var AB = vector(r.A, r.B);
    var AB = vRB.clone().sub(vRA);
    //var AM = vector(r.A, m);
    var AM = vP.clone().sub(vRA);
    //var BC = vector(r.B, r.C);
    var BC = vRC.clone().sub(vRB);
    //var BM = vector(r.B, m);
    var BM = vP.clone().sub(vRB);

    var dotABAM = AB.dot(AM);
    var dotABAB = AB.dot(AB);
    var dotBCBM = BC.dot(BM);
    var dotBCBC = BC.dot(BC);

    return 0 <= dotABAM && dotABAM <= dotABAB && 0 <= dotBCBM && dotBCBM <= dotBCBC;
}

function move(tickturn, perception) {

    let followpos = new Vector2(0, perception.Specs.MaxSpeed/3);
    let desired = followpos.clone();
    let steering = desired.clone();

    const agentvelocity = Vector2.fromArray(perception.Internal.Velocity);
    const agentradius = perception.Internal.Proprioception;
    const visionradius = perception.Specs.VisionRadius;

    let debugpoints = [];

    // on évite les obstacles
    let avoidanceforce = new Vector2();
    if(perception.External.Vision) {

        // normals

        const normals = agentvelocity.normals();
        const bottomleft = normals[0].clone().mag(agentradius+10);
        const bottomright = normals[1].clone().mag(agentradius+10);
        
        // bordures couloir gauche et droite

        const topleft = bottomleft.clone().rotate(-Math.PI/2).mag(visionradius+5).add(bottomleft);
        const topright = bottomright.clone().rotate(Math.PI/2).mag(visionradius+5).add(bottomright);

        // topcap

        //const topcap = rightedge.clone().sub(leftedge);

        for(const otheragentkey in perception.External.Vision) {
            const otheragent = perception.External.Vision[otheragentkey];
            if(otheragent.Tag !== "obstacle") continue;

            closeEdge = Vector2.fromArray(otheragent.CloseEdge);
            farEdge = Vector2.fromArray(otheragent.FarEdge);
            const segment = farEdge.clone().sub(closeEdge);

            const edgestoavoid = [];

            // can be overlapping, but not sure yet (test has been made on an axis aligned bounding box, but the corridor is oriented, not aligned)

            if(pointInRectangle(closeEdge, bottomleft, bottomright, topright, topleft)) {
                //console.log(closeEdge, "CLOSEEDGE IN CORRIDOR !");
                edgestoavoid.push(closeEdge);
            }

            if(pointInRectangle(farEdge, bottomleft, bottomright, topright, topleft)) {
                //console.log(farEdge, "FAREDGE IN CORRIDOR !");
                //debugpoints.push(farEdge);
                edgestoavoid.push(farEdge);
            }

            // test corridor intersection with line segment
            let collision = Vector2.intersectionWithLineSegment(bottomleft, topleft, closeEdge, farEdge);
            if(collision.intersects && !collision.colinear) {
                // COLLISION LEFT
                edgestoavoid.push(collision.intersection);
            }

            collision = Vector2.intersectionWithLineSegment(bottomright, topright, closeEdge, farEdge);
            if(collision.intersects && !collision.colinear) {
                // COLLISION RIGHT
                edgestoavoid.push(collision.intersection);
            }

            if(edgestoavoid.length > 0) {
                if(edgestoavoid.length !== 2) {
                    console.log("OBSTACLE, SUM THIN WONG !");
                } else {
                    const pointa = edgestoavoid[0];
                    const pointb = edgestoavoid[1];
                    const center = pointb.clone().add(pointa).div(2);
                    debugpoints.push(center);

                    avoidanceforce.sub(new Vector2(100, 0));
                }
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
    ].concat(debugpoints.map(point => {
        return { Method: 'debugvis', Arguments: [point.toArray(5)] };
    })))
    .catch(err => { throw err; });

}

function RadianToDegree(rad) {
    return rad * (180.0 / Math.PI);
}

comm.connect(port, host, agentid)
.then(function({ sendRequest, sendMutations, onTick }) {
    onTick(move.bind({ sendRequest, sendMutations }));
});
