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
    $(renderer.view).on('mousemove', function() {
        const mouseData = renderer.plugins.interaction.mouse.global;
        $('#infos').html('(' + mouseData.x + ', ' + (renderer.height - mouseData.y) + ']');
    });

    $debug = $('#debug');

    var stage = new PIXI.Container();
    stage.position.y = renderer.height / renderer.resolution;
    stage.scale.y = -1;

    const agenttexture = PIXI.loader.resources["images/triangle.png"].texture;
    agenttexture.rotate = 8;

    window.onStateUpdate = function(points) {
        stage.removeChildren();

        const debug = $debug.is(':checked');

        if (points.Obstacles) {
            const obstacles = (new PIXI.Graphics())
                .lineStyle(3, 0x00FF00);
            stage.addChild(obstacles);

            points.Obstacles.forEach((obstacle) => {
                obstacles
                    .moveTo(obstacle.A[0], obstacle.A[1])
                    .lineTo(obstacle.B[0], obstacle.B[1]);
            });
        }

        if (debug && points.DebugIntersects) {
            const intersects = new PIXI.Graphics();
            stage.addChild(intersects);

            points.DebugIntersects.forEach((intersect) => {
                intersects
                    .beginFill(0x0000FF)
                    .drawCircle(intersect[0], intersect[1], 3)
                    .endFill();
            });
        }

        if (debug && points.DebugIntersectsRejected) {
            const intersects = new PIXI.Graphics();
            stage.addChild(intersects);

            points.DebugIntersectsRejected.forEach((intersect) => {
                intersects
                    .beginFill(0xFF0000)
                    .drawCircle(intersect[0], intersect[1], 3)
                    .endFill();
            });
        }

        if (points.Projectiles) {
            const projectiles = (new PIXI.Graphics())
                    .lineStyle(2, 0xFF0000);
            stage.addChild(projectiles);

            points.Projectiles.forEach((projectile) => {
                projectiles
                    .moveTo(projectile.From.Position[0], projectile.From.Position[1])
                    .lineTo(projectile.Position[0], projectile.Position[1]);
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

                if(debug) createAgentVision(agent).drawInStage(stage);

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