// create shader
// https://playcanvas.com/editor/code/497594?tabs=8741163,8741167

// billboards in playcanvs
// https://playcanvas.com/editor/code/362231?tabs=7893846

// Fly camera
// https://playcanvas.com/editor/code/362231?tabs=7893846,7894087

const textobj = pc.app.scene.root.children[3].children[0];

const camera = pc.app.root.findByName("Camera");
const box = pc.app.root.findByName("Box");
const screen2d = pc.app.root.findByName("2D Screen");

screen2d.screen.scaleMode = pc.SCALEMODE_NONE;

const entity = new pc.Entity();
entity.addComponent("element", { type: "text" });
entity.setPosition(new pc.Vec3(0, 0, 0));
entity.setLocalPosition(new pc.Vec3(0, 0, 0));
entity.element.text = "Hello, World!";
entity.element.font = textobj.element.font;
entity.element.pivot = new pc.Vec2(0.5, 0.5);
entity.element.anchor = new pc.Vec4(0.0, 0.0, 0.0, 0.0);
screen2d.addChild(entity);

const mapPos = function(worldCoords) {
    const res = screen2d.screen.referenceResolution;
    const screenCoords = camera.camera.worldToScreen(worldCoords);
    return new pc.Vec3(
        screenCoords.x,
        res.y - screenCoords.y,
        0
    );
};

const reposition = function () {
    entity.setLocalPosition(mapPos(box.getLocalPosition()));
};

window.addEventListener("resize", reposition);

let i = 0;

const raf = function() {
    reposition();
    const x = Math.cos(i/100);
    const y = Math.sin(i/100);
    //console.log(x, y);
    box.setLocalPosition(x, y, 0);
    window.requestAnimationFrame(raf);

    i++;
};

raf();
