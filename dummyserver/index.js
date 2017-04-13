const dgram = require('dgram');
const now = require('performance-now');
const server = dgram.createSocket('udp4');
const chalk = require('chalk');

const _clients = [];
const pending = {};
let ticking = false;
let turn = 0;
let sent = 0;

server.on('error', (err) => {
  console.log(`server error:\n${err.stack}`);
  server.close();
});

const guys = 100;
const tps = 100;
const msPerTurn = 1000 / tps;
const times = [];
const timeouts = [];

function measurespeed(duration) {
  const timeout = msPerTurn * 0.60;

  if (duration >= timeout) {
    timeouts[sent % (tps * guys)] = 1;
  } else {
    timeouts[sent % (tps * guys)] = 0;
  }

  times[sent % (tps * guys)] = duration;
}

server.on('message', (msg, rinfo) => {
  const json = JSON.parse(msg);

  // (turn % 100 === 0) && process.stdout.write('.');

  if (json.Type === 'Handshake') {
    _clients.push(Object.assign({}, rinfo, {AgentId: json.AgentId}));

    if (!ticking && _clients.length === guys) {
      startTicking();
    }
  } else {
    const currentTurnPayload = pending[`${json.AgentId}${json.Payload.Turn}`];
    const diff = (now() - currentTurnPayload);

    measurespeed(diff);

    // if (diff >= timeout) {
    //   process.stdout.write('\n');
    //   console.log(chalk.red(`Agent:${json.AgentId} turn:${json.Payload.Turn}`), diff.toFixed(2), 'ms');
    // }
  }
});

server.on('listening', () => {
  const address = server.address();
  console.log(`server listening ${address.address}:${address.port}`);
});

server.bind(8888);

function startTicking() {
  ticking = true;

  setInterval(function() {

    const mean = times.reduce(function(carry, val) {
      return carry + val;
    }, 0) / times.length;

    const squareDists = times.map((a) => {
      return Math.pow(a - mean, 2);
    });

    const squareDistsSum = squareDists.reduce((a, b) => a + b, 0);
    const timeoutsSum = timeouts.reduce((a, b) => a + b, 0);

    const stddev = Math.sqrt(squareDistsSum / times.length);

    console.log(
      'Mean:', mean.toFixed(2),
      ' Timeouts per sec:', timeoutsSum,
      ' std dev:', stddev,
      ' from elements:', timeouts.length
    );
  }, 1000);

  setInterval(() => {

    _clients.forEach((c) => {
      sent++;
      const msg = JSON.stringify({
        Method: 'tick',
      });

      const buffer = new Buffer(msg);

      pending[`${c.AgentId}${turn}`] = now();

      server.send(buffer, 0, buffer.length, c.port, c.address, (err) => {
        if (err) throw err;
      });
    });

    turn++;

    (turn % tps === 0) && console.log(turn, sent);
  }, msPerTurn);
}
