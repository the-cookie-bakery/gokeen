#version 450

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inUV0;
layout(location = 3) in vec2 inUV1;
layout(location = 4) in vec4 inColor;

layout(set = 0, binding = 0) uniform SceneUniforms {
    mat4 viewProjection;
    vec4 cameraPosition;
    vec4 lightDirection;
    vec4 lightColor;
    vec4 ambientColor;
} scene;

layout(push_constant) uniform MaterialPush {
    mat4 model;
    vec4 baseColor;
    vec4 parameters;
} material;

layout(location = 0) out vec3 vNormal;
layout(location = 1) out vec2 vUV0;
layout(location = 2) out vec2 vUV1;
layout(location = 3) out vec4 vColor;
layout(location = 4) out vec3 vWorldPosition;

void main() {
    vec4 world = material.model * vec4(inPosition, 1.0);
    gl_Position = scene.viewProjection * world;
    vNormal = normalize(mat3(material.model) * inNormal);
    vUV0 = inUV0;
    vUV1 = inUV1;
    vColor = inColor;
    vWorldPosition = world.xyz;
}
