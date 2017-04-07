const comm = require('./utils/comm');
const Vector2 = require('./utils/vector2');
const {map} = require('./utils/calc');

process.on('SIGTERM', () => process.exit());

const port = process.env.SWARMPORT;
const host = process.env.SWARMHOST;
const agentid = process.env.AGENTID;

function move(tickturn, perception) {
  const attractorpos = Vector2.fromArray(perception.Objective.Attractor);

  let v = 0
  let sum = [];
  for(k = 0; k < 20000; k++) {
      sum.push(v++);
  }

  this.sendMutations(tickturn, [
      { Method: 'shoot', Arguments: attractorpos.toArray(5), Sum: sum.reduce(function(carry, v) { return v + carry; }, 0) },
  ])
  .catch((err) => { throw err; });
}

comm.connect(port, host, agentid)
.then(function({sendRequest, sendMutations, onTick}) {
  onTick(move.bind({sendRequest, sendMutations}));
});
