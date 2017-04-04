const comm = require('./utils/comm');
const Vector2 = require('./utils/vector2');
const {map} = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

function move(tickturn, perception) {
  const attractorpos = Vector2.fromArray(perception.Objective.Attractor);

  this.sendMutations(tickturn, [
      ['mutationShoot', attractorpos.toArray(5)],
  ])
  .catch((err) => { throw err; });
}

comm.connect(port, host, agentid)
.then(function({sendRequest, sendMutations, onTick}) {
  onTick(move.bind({sendRequest, sendMutations}));
});
