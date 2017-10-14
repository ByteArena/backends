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