// create shader
// https://playcanvas.com/editor/code/497594?tabs=8741163,8741167

// billboards in playcanvs
// https://playcanvas.com/editor/code/362231?tabs=7893846

// Fly camera
// https://playcanvas.com/editor/code/362231?tabs=7893846,7894087

const textobj = pc.app.scene.root.children[3].children[0];
textobj.element.text = "ddd";
textobj.setLocalPosition(-20, 20, 0);

const t2 = textobj.clone();
t2.element.text = "noooooo";
t2.setLocalPosition(-10, 0, 0);
textobj.parent.addChild(t2);

// https://developer.playcanvas.com/en/api/pc.CameraComponent.html#worldToScreen

// var div = document.createElement("div");
// div.innerHTML = "Capsule";
// div.style.fontFamily = "Verdana, sans-serif";
// div.style.color = "#fff";
// div.style.position = "absolute";
// document.body.appendChild(div);
// let pos = camera.camera.worldToScreen(box.getPosition());
// div.style.left = pos.x + "px";
// div.style.top = pos.y + "px";

const camera = pc.app.root.findByName("Camera");
const box = pc.app.root.findByName("Box");
const screen2d = pc.app.root.findByName("2D Screen");

screen2d.screen.scaleMode = pc.SCALEMODE_NONE;

const mapPos = function(screenCoords) {
    const res = screen2d.screen.referenceResolution;
    return new pc.Vec3(
        screenCoords.x - 0.5 * res.x,
        screenCoords.y - 0.5 * res.y,
        0
    );
};

window.addEventListener("resize", function() {
    window.requestAnimationFrame(function() {
        t2.setLocalPosition(
            mapPos(camera.camera.worldToScreen(box.getPosition()))
        );
    });
});
