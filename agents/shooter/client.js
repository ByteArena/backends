const comm = require('./utils/comm');
const Vector2 = require('./utils/vector2');
const {map} = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

function move(tickturn, perception) {

  // Finding the attractor
  let attractor = null;
  for(const otheragentkey in perception.External.Vision) {
    const otheragent = perception.External.Vision[otheragentkey]
    if(otheragent.Tag === "attractor") {
      attractor = otheragent;
      break;
    }
  }

  if(attractor === null) return;

  const attractorpos = Vector2.fromArray(attractor.Center);
  const attractorvelocity = Vector2.fromArray(attractor.Velocity);
  const aimed = attractorpos.add(attractorvelocity)

  if(Math.random() >= .9) {
      this.sendMutations(tickturn, [
        {
          Method: 'shoot',
          Arguments: aimed.toArray(5)
        },
      ])
      .catch((err) => { throw err; });
  }
}

comm.connect(port, host, agentid)
.then(function({sendRequest, sendMutations, onTick}) {
  onTick(move.bind({sendRequest, sendMutations}));
});
