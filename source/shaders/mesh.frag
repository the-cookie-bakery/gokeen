#version 450

layout(location = 0) in vec3 vNormal;
layout(location = 1) in vec2 vUV0;
layout(location = 2) in vec2 vUV1;
layout(location = 3) in vec4 vColor;
layout(location = 4) in vec3 vWorldPosition;

layout(set = 0, binding = 0) uniform SceneUniforms {
    mat4 viewProjection;
    vec4 cameraPosition;
    vec4 lightDirection;
    vec4 lightColor;
    vec4 ambientColor;
} scene;

layout(set = 0, binding = 1) uniform sampler2D baseTexture;
layout(set = 0, binding = 2) uniform sampler2D lightmapTexture;

layout(push_constant) uniform MaterialPush {
    mat4 model;
    vec4 baseColor;
    vec4 parameters;
} material;

layout(location = 0) out vec4 outColor;

void main() {
    vec4 base = texture(baseTexture, vUV0) * material.baseColor * vColor;
    vec3 lightmap = vec3(1.0);
    if (material.parameters.x > 0.5) {
        lightmap = texture(lightmapTexture, vUV1).rgb;
    }

    vec3 normal = normalize(vNormal);
    vec3 lightDir = normalize(-scene.lightDirection.xyz);
    vec3 viewDir = normalize(scene.cameraPosition.xyz - vWorldPosition);
    vec3 halfDir = normalize(lightDir + viewDir);

    float diffuse = max(dot(normal, lightDir), 0.0);
    float shininess = max(material.parameters.y, 1.0);
    float specular = pow(max(dot(normal, halfDir), 0.0), shininess) * 0.25;

    vec3 ambient = scene.ambientColor.rgb * lightmap;
    vec3 direct = scene.lightColor.rgb * diffuse * lightmap;
    vec3 spec = scene.lightColor.rgb * specular * lightmap;
    vec3 color = base.rgb * (ambient + direct) + spec * base.rgb;

    outColor = vec4(color, base.a);
}
