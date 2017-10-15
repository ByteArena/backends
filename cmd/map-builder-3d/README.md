# Playcanvas JSON

model
    .version
	.nodes[nodeindex]
		.name
		.position
		.rotation
		.scale
	.parents[]
	.vertices[verticesindex]
		.position[]
		.normal
	.meshes[meshindex]
		.aabb
		.vertices: verticesindex
		.indices
		.type: "triangles"
		.base: ?
		.count: 
	.meshInstances[]
		.node: nodeindex
		.mesh: meshindex

# FBX dump

.children[name="Objects"]
    .children[name="Geometry"][]
        .properties[]
            .0: id numérique de la géométrie
            .1: chaîne terminée par \null: nom de la géométrie
            .2: type (mesh)
        .children[name="Vertices"]
            .properties[]
                .0
                    .value: array of floats
        .children[name="PolygonVertexIndex"]
            .properties[]
                .0
                    .value: array of floats (empty)
    .children[name="Model"][]
        .properties[]
            .0: id numérique du modèle
            .1: chaîne terminée par \null: nom du modèle
            .2: type (mesh)
        .children[name="Properties70"]
            => transform
.children[name="Connections"]
    .children["name=C"][]
        .properties
            .0: type (chaîne "OO")
            .1 et .2: si .2 != 0 => .1 : id numérique géométrie, .2 : id numérique modèle