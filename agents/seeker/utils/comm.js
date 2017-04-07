const dgram = require('dgram');

module.exports = {
    connect: function connect(port, host, agentid) {

        const wrapInTransport = function(type, payload) {
            return {
                AgentId: agentid,
                Type: type,
                Payload: payload,
            };
        };

        const p = new Promise(function(resolve, reject) {

            const client = dgram.createSocket('udp4');
            const json = JSON.stringify({
                AgentId: agentid,
                Type: "Handshake",
                Payload: {
                    Greetings: 'Hello from ' + agentid + ' !'
                }
            });

            const message = new Buffer(json);

            client.send(message, 0, message.length, port, host, function(err, nbbytes) {

                // handshake successful

                let cbktickrequested = function() {}; // no-op

                client.on('message', function(data, rinfo) {
                    const json = data.toString();
                    const decoded = JSON.parse(json);

                    if('Method' in decoded) {
                        // Request emitted by server; not handling session yet (one way messaging, like pubsub)
                        if(decoded.Method === 'tick') {
                            const tickturn = parseInt(decoded.Arguments[0]);
                            const senses = decoded.Arguments[1];
                            cbktickrequested(tickturn, senses);
                        } else {
                            throw new Error('Undefined Method requested from server : ' + decoded.Method);
                        }
                    } else {
                        throw new Error('Invalid message received from server :' + json);
                    }
                });

                resolve({
                    sendMutations(turn, mutations) {
                        return new Promise(function(resolve, reject) {
                            const json = JSON.stringify(wrapInTransport('Mutation', {
                                Turn: turn,
                                Mutations: mutations
                            }));

                            client.send(json, 0, json.length, port, host, function(err, nbbytes) {
                                resolve();
                            });
                        });
                    },
                    onTick(cbk) {
                        cbktickrequested = cbk;
                    }
                });
            });
        });

        return p;
    }
}
