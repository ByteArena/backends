const net = require('net');
const readline = require('readline');

module.exports = {
    connect: function connect(port, host, agentid) {

        const p = new Promise(function(resolve, reject) {

            const client = new net.Socket();
            client.on('close', function() {
                console.log('Connection closed');
            });

            client.connect(port, host, function() {

                client.write(JSON.stringify({
                    Agent: agentid,
                    Handshake: 'hello from ' + agentid + ' !'
                }) + '\n', 'UTF8', function() {

                    // handshake successful

                    let reqid = 0;
                    const pending = {};
                    let cbktickrequested = function() {}; // no-op

                    var i = readline.createInterface(client, client);
                    i.on('line', function(data) {
                        const json = data.toString();
                        const decoded = JSON.parse(json);

                        if('RequestId' in decoded) {

                            // Response to one of our requests
                            if(decoded.RequestId in pending) {
                                pending[decoded.RequestId][0](decoded.Results);
                                pending[decoded.RequestId] = null;
                                delete pending[decoded.RequestId];
                            } else {
                                throw new Error('Undefined ResponseId : ' + decoded.RequestId);
                            }
                        } else if('Method' in decoded) {
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
                        sendMutations(/*tickturn, */...args) {
                            client.write(JSON.stringify({
                                Agent: agentid,
                                Method: 'mutations',
                                Arguments: args
                            }) + '\n');

                            return Promise.resolve();
                        },
                        sendRequest(method, ...args) {
                            const thisid = reqid++;
                            client.write(JSON.stringify({
                                Agent: agentid,
                                Method: method,
                                Arguments: args || [],
                                RequestId: thisid
                            }) + '\n');

                            return new Promise(function(resolve, reject) {
                                pending[thisid] = [resolve, reject];
                            });
                        },
                        onTick(cbk) {
                            cbktickrequested = cbk;
                        }
                    });
                });
            });
        });

        return p;
    }
}
