/* global PIXI */
$(function() {
    var $output = $("#output");
    var $open = $("#open");
    var $close = $("#close");
    var ws;

    var print = function(message) {
        var d = $("<div></div>");
        d.html(message);
        output.append(d.get(0));
    };

    $open.click(function(e) {

        if (ws) return;

        ws = new WebSocket("{{.}}");

        ws.onopen = function(evt) {
            print("OPEN");
            $open.hide();
            $close.show();
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws.close();
            ws = null;
            $open.show();
            $close.hide();
        }

        ws.onmessage = function(evt) {
            window.onStateUpdate(JSON.parse(evt.data));
        }

        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
    });

    $close.click(function(e) {
        if (!ws) return;
        ws.close();
    });

});

function createAgentVision(agent) {
    const vision = new PIXI.Graphics();

    const agentposition = Vector2.fromArray(agent.Position);
    const radius = agent.VisionRadius;
    const angle = agent.VisionAngle;
    const orientation = agent.Orientation;

    const halfangle = angle/2;

    const leftlineto = (new Vector2(1, 1))
        .mag(radius)
        .setAngle(orientation - halfangle);
    
    const rightlineto = (new Vector2(1, 1))
        .mag(radius)
        .setAngle(orientation + halfangle);

    vision.lineStyle(1, 0xFFAAFF);

    vision.arc(
        agentposition.x,
        agentposition.y,
        radius,
        (Math.PI/2) + (-1*orientation) - halfangle,
        (Math.PI/2) + (-1*orientation) + halfangle
    );


    vision
        .moveTo(agentposition.x, agentposition.y)
        .lineTo(agentposition.x+leftlineto.x, agentposition.y+leftlineto.y);
    
    vision
        .moveTo(agentposition.x, agentposition.y)
        .lineTo(agentposition.x+rightlineto.x, agentposition.y+rightlineto.y);

    return {
        drawInStage(stage) {
            stage.addChild(vision);
        }
    };
}

function render() {

    //Create the renderer
    var renderer = PIXI.autoDetectRenderer(1000, 600, {
        antialias: true
    });
    renderer.backgroundColor = 0xFFFFFF;
    $('#visualization').append(renderer.view);

    var stage = new PIXI.Container();
    stage.position.y = renderer.height / renderer.resolution;
    stage.scale.y = -1;

    const agenttexture = PIXI.loader.resources["images/triangle.png"].texture;
    agenttexture.rotate = 8;

    window.onStateUpdate = function(points) {
        stage.removeChildren();

        if (points.Projectiles) {
            points.Projectiles.forEach((projectile) => {
                const position = Vector2.fromArray(projectile.Position);
                const positionfrom = Vector2.fromArray(projectile.From.Position);
                const line = new PIXI.Graphics();

                line
                    .lineStyle(2, 0xFF0000)
                    .moveTo(positionfrom.x, positionfrom.y)
                    .lineTo(position.x, position.y);

                stage.addChild(line);
            });
        }

        if (points.Agents) {
            points.Agents.forEach((agent) => {

                var sprite = new PIXI.Sprite(agenttexture);

                const position = Vector2.fromArray(agent.Position);

                sprite.x = position.x;
                sprite.y = position.y;
                sprite.width = agent.Radius * 2;
                sprite.height = agent.Radius * 2;
                sprite.tint = 0x8D8D64;
                sprite.anchor.set(0.5);
                sprite.rotation = -1 * agent.Orientation

                createAgentVision(agent).drawInStage(stage);

                //Create a container object called the `stage`
                stage.addChild(sprite);
            });
        }

        window.requestAnimationFrame(() => renderer.render(stage));
    };
}

PIXI
    .loader
    .add('images/triangle.png')
    .load(render);