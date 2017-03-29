const comm = require('./comm');
const now = require('performance-now');

process.on('SIGTERM', () => {
    process.exit();
});

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

function move(tickturn, senses) {
    console.log(senses);
    const start = now();
    this.sendRequest('getGreetings', 'jérôme')
        .then(results => {
            return this.sendMutations(tickturn, [
                ['mutationIncrement'],
                ['mutationAccelerate', [0.8, 0.5]],
            ]);
        })
        .then(response => {
            measurespeed(start);
        })
        .catch(err => { throw err; });
}

comm.connect(port, host, agentid)
.then(function({ sendRequest, sendMutations, onTick }) {
    onTick(move.bind({ sendRequest, sendMutations }));
});
