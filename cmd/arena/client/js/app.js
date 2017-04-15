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

  // Divided by 10 for now to stay in the stage
  const radius = agent.VisionRadius;

  vision.lineStyle(2, 0xFF00FF);
  vision.drawCircle(agent.X, agent.Y, radius);

  vision.endFill();

  return {
    drawInStage(stage) {
      stage.addChild(vision);
    }
  };
}

function render() {

  //Create the renderer
  var renderer = PIXI.autoDetectRenderer(1000, 600, { antialias: true });
  renderer.backgroundColor = 0xFFFFFF;
  $('#visualization').append(renderer.view);

  var stage = new PIXI.Container();

  window.onStateUpdate = function(points) {
    stage.removeChildren();

    if (points.Projectiles) {
      points.Projectiles.forEach((projectile) => {
        const line = new PIXI.Graphics();

        line
          .lineStyle(2, 0xFF0000)
          .moveTo(projectile.From.X, projectile.From.Y)
          .lineTo(projectile.X, projectile.Y);

        stage.addChild(line);
      });
    }

    if (points.Agents) {
      points.Agents.forEach((agent) => {

        var sprite = new PIXI.Sprite(
          PIXI.loader.resources['images/triangle.png'].texture
        );

        sprite.x = agent.X;
        sprite.y = agent.Y;
        sprite.width = agent.Radius * 2;
        sprite.height = agent.Radius * 2;
        sprite.tint = 0x8D8D64;
        sprite.anchor.set(0.5);
        sprite.rotation = Math.PI / 2 + agent.Orientation; // Math.PI/2: quart de tour vers la droite

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
  .add('images/circle.png')
  .add('images/triangle.png')
  .load(render);
