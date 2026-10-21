#version 450

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inUV0;
layout(location = 3) in vec2 inUV1;
layout(location = 4) in vec4 inColor;
layout(location = 5) in uvec4 inJoints;
layout(location = 6) in vec4 inWeights;

layout(set = 0, binding = 0) uniform SceneUniforms {
    mat4 viewProjection;
    vec4 cameraPosition;
    vec4 lightDirection;
    vec4 lightColor;
    vec4 ambientColor;
} scene;

layout(set = 0, binding = 1) uniform sampler2D baseTexture;
layout(set = 0, binding = 2) uniform sampler2D lightmapTexture;

layout(set = 1, binding = 0) uniform BonePalette {
    mat4 bones[64];
} palette;

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
    mat4 skin = palette.bones[inJoints.x] * inWeights.x
              + palette.bones[inJoints.y] * inWeights.y
              + palette.bones[inJoints.z] * inWeights.z
              + palette.bones[inJoints.w] * inWeights.w;
    vec4 local = skin * vec4(inPosition, 1.0);
    vec4 world = material.model * local;
    gl_Position = scene.viewProjection * world;
    vNormal = normalize(mat3(material.model) * mat3(skin) * inNormal);
    vUV0 = inUV0;
    vUV1 = inUV1;
    vColor = inColor;
    vWorldPosition = world.xyz;
}
