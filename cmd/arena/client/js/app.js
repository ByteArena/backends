/* global PIXI, $, bytearenasdk */
(function($, PIXI, Vector2) {
    function createAgentVision(agent) {
        const vision = new PIXI.Graphics();

        const agentposition = Vector2.fromArray(agent.Position);
        const agentvelocity = Vector2.fromArray(agent.Velocity);
        const agentradius = agent.Radius;

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

        // normals
        const normals = agentvelocity.normals();
        const normalegauche = normals[0].clone().mag(agentradius+agentradius);
        const normaledroite = normals[1].clone().mag(agentradius+agentradius);

        vision
            .lineStyle(2, 0xFF0000)
            .moveTo(agentposition.x, agentposition.y)
            .lineTo(agentposition.x+normalegauche.x, agentposition.y+normalegauche.y);
        
        vision
            .lineStyle(2, 0x0000FF)
            .moveTo(agentposition.x, agentposition.y)
            .lineTo(agentposition.x+normaledroite.x, agentposition.y+normaledroite.y);
        
        // bordures couloir gauche et droite

        const leftedge = normalegauche.clone().rotate(-Math.PI/2).mag(radius+5).add(normalegauche);
        const rightedge = normaledroite.clone().rotate(Math.PI/2).mag(radius+5).add(normaledroite);

        vision
            .lineStyle(2, 0xFF0000)
            .moveTo(agentposition.x+normalegauche.x, agentposition.y+normalegauche.y)
            .lineTo(agentposition.x+leftedge.x, agentposition.y+leftedge.y);
        
        vision
            .lineStyle(2, 0x0000FF)
            .moveTo(agentposition.x+normaledroite.x, agentposition.y+normaledroite.y)
            .lineTo(agentposition.x+rightedge.x, agentposition.y+rightedge.y);
        
        // topcap

        const topcap = rightedge.clone().sub(leftedge);
        vision
            .lineStyle(2, 0x000000)
            .moveTo(agentposition.x+leftedge.x, agentposition.y+leftedge.y)
            .lineTo(agentposition.x+leftedge.x+topcap.x, agentposition.y+leftedge.y+topcap.y);

        return {
            drawInStage(stage) {
                stage.addChild(vision);
            }
        };
    }

    function render(arenawidth, arenaheight) {

        console.log(arenawidth, arenaheight);

        //Create the renderer
        var renderer = PIXI.autoDetectRenderer(arenawidth, arenaheight, {
            antialias: true
        });
        renderer.backgroundColor = 0xFFFFFF;
        $('#visualization').append(renderer.view);
        $(renderer.view).on('mousemove', function() {
            const mouseData = renderer.plugins.interaction.mouse.global;
            $('#infos').html('(' + Math.round(mouseData.x) + ', ' + Math.round(renderer.height - mouseData.y) + ')');
        });

        $debug = $('#debug');

        var stage = new PIXI.Container();
        stage.position.y = renderer.height / renderer.resolution;
        stage.scale.y = -1;

        const agenttexture = PIXI.loader.resources["images/triangle.png"].texture;
        agenttexture.rotate = 8;

        $("html").on("bytearena:stateupdate", function(evt, points) {

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

            if (debug && points.DebugPoints) {
                const layer = new PIXI.Graphics();
                stage.addChild(layer);
                points.DebugPoints.forEach((point) => {
                    layer
                        .beginFill(0xFF0000)
                        .drawCircle(point[0], point[1], 3)
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
        });
    }

    window.start = function(arenawidth, arenaheight) {
        PIXI
        .loader
        .add('images/triangle.png')
        .load(render.bind(null, arenawidth, arenaheight));
    };
})($, PIXI, bytearenasdk.vector.Vector2)