// cursor-fluid.ts texture passes. Texture coordinates here increase from the
// simulation's bottom edge; only the display pass flips rows for native RGBA.
Texture2D<float4> source : register(t0);
Texture2D<float4> auxiliary : register(t1);
SamplerState linearClamp : register(s0);
SamplerState nearestClamp : register(s1);
cbuffer FluidParameters : register(b0) {
  float4 grid; // texelSize.xy, dt, dissipation
  float4 splat; // point.xy, radius, aspect
  float4 ink;
};
struct Vertex { float4 position : SV_Position; float2 uv : TEXCOORD0; };
Vertex fluidVS(uint id : SV_VertexID) {
  Vertex v;
  v.uv = float2((id << 1) & 2, id & 2);
  v.position = float4(v.uv.x * 2 - 1, 1 - v.uv.y * 2, 0, 1);
  return v;
}
float4 copyPS(Vertex v) : SV_Target { return source.SampleLevel(linearClamp, v.uv, 0); }
float4 splatPS(Vertex v) : SV_Target {
  float2 p = v.uv - splat.xy; p.x *= splat.w;
  return float4(source.SampleLevel(linearClamp, v.uv, 0).rgb + exp(-dot(p,p)/splat.z)*ink.rgb, 1);
}
float4 curlPS(Vertex v) : SV_Target {
  float L=source.SampleLevel(nearestClamp,v.uv-float2(grid.x,0),0).y;
  float R=source.SampleLevel(nearestClamp,v.uv+float2(grid.x,0),0).y;
  float T=source.SampleLevel(nearestClamp,v.uv+float2(0,grid.y),0).x;
  float B=source.SampleLevel(nearestClamp,v.uv-float2(0,grid.y),0).x;
  return float4(.5*(R-L-T+B),0,0,1);
}
float4 vorticityPS(Vertex v) : SV_Target {
  float L=auxiliary.SampleLevel(nearestClamp,v.uv-float2(grid.x,0),0).x;
  float R=auxiliary.SampleLevel(nearestClamp,v.uv+float2(grid.x,0),0).x;
  float T=auxiliary.SampleLevel(nearestClamp,v.uv+float2(0,grid.y),0).x;
  float B=auxiliary.SampleLevel(nearestClamp,v.uv-float2(0,grid.y),0).x;
  float C=auxiliary.SampleLevel(nearestClamp,v.uv,0).x;
  float2 force=.5*float2(abs(T)-abs(B),abs(R)-abs(L));
  force=force/(length(force)+.0001)*3*C; force.y*=-1;
  return float4(clamp(source.SampleLevel(linearClamp,v.uv,0).xy+force*grid.z,-1000,1000),0,1);
}
float4 divergencePS(Vertex v) : SV_Target {
  float L=source.SampleLevel(nearestClamp,v.uv-float2(grid.x,0),0).x;
  float R=source.SampleLevel(nearestClamp,v.uv+float2(grid.x,0),0).x;
  float T=source.SampleLevel(nearestClamp,v.uv+float2(0,grid.y),0).y;
  float B=source.SampleLevel(nearestClamp,v.uv-float2(0,grid.y),0).y;
  float2 C=source.SampleLevel(nearestClamp,v.uv,0).xy;
  if(v.uv.x-grid.x<0) L=-C.x; if(v.uv.x+grid.x>1) R=-C.x;
  if(v.uv.y+grid.y>1) T=-C.y; if(v.uv.y-grid.y<0) B=-C.y;
  return float4(.5*(R-L+T-B),0,0,1);
}
float4 clearPS(Vertex v) : SV_Target { return .1*source.SampleLevel(nearestClamp,v.uv,0); }
float4 pressurePS(Vertex v) : SV_Target {
  float L=source.SampleLevel(nearestClamp,v.uv-float2(grid.x,0),0).x;
  float R=source.SampleLevel(nearestClamp,v.uv+float2(grid.x,0),0).x;
  float T=source.SampleLevel(nearestClamp,v.uv+float2(0,grid.y),0).x;
  float B=source.SampleLevel(nearestClamp,v.uv-float2(0,grid.y),0).x;
  return float4((L+R+T+B-auxiliary.SampleLevel(nearestClamp,v.uv,0).x)*.25,0,0,1);
}
float4 gradientPS(Vertex v) : SV_Target {
  float L=auxiliary.SampleLevel(nearestClamp,v.uv-float2(grid.x,0),0).x;
  float R=auxiliary.SampleLevel(nearestClamp,v.uv+float2(grid.x,0),0).x;
  float T=auxiliary.SampleLevel(nearestClamp,v.uv+float2(0,grid.y),0).x;
  float B=auxiliary.SampleLevel(nearestClamp,v.uv-float2(0,grid.y),0).x;
  return float4(source.SampleLevel(linearClamp,v.uv,0).xy-float2(R-L,T-B),0,1);
}
float4 advectionPS(Vertex v) : SV_Target {
  float2 velocity=auxiliary.SampleLevel(linearClamp,v.uv,0).xy;
  return source.SampleLevel(linearClamp,v.uv-grid.z*velocity*grid.xy,0)/(1+grid.w*grid.z);
}
float4 displayPS(Vertex v) : SV_Target {
  float2 uv=float2(v.uv.x,1-v.uv.y);
  float3 c=source.SampleLevel(linearClamp,uv,0).rgb;
  float dx=length(source.SampleLevel(linearClamp,uv+float2(grid.x,0),0).rgb)-length(source.SampleLevel(linearClamp,uv-float2(grid.x,0),0).rgb);
  float dy=length(source.SampleLevel(linearClamp,uv+float2(0,grid.y),0).rgb)-length(source.SampleLevel(linearClamp,uv-float2(0,grid.y),0).rgb);
  c*=clamp(normalize(float3(dx,dy,length(grid.xy))).z+.7,.7,1);
  return float4(c,max(c.r,max(c.g,c.b)));
}
