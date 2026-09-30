import { useRef } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';

function AbstractWireframe() {
  const mesh1 = useRef<THREE.Mesh>(null);
  const mesh2 = useRef<THREE.Mesh>(null);

  useFrame((state) => {
    const t = state.clock.getElapsedTime();
    if (mesh1.current && mesh2.current) {
      mesh1.current.rotation.y = t * 0.05;
      mesh1.current.rotation.x = t * 0.07;
      
      mesh2.current.rotation.y = t * -0.04;
      mesh2.current.rotation.z = t * 0.03;
    }
  });

  return (
    <group position={[0, 0, -2]}>
      <mesh ref={mesh1} scale={4}>
        <icosahedronGeometry args={[1, 2]} />
        <meshBasicMaterial color="#10b981" wireframe transparent opacity={0.08} />
      </mesh>
      <mesh ref={mesh2} scale={4.5}>
        <icosahedronGeometry args={[1, 1]} />
        <meshBasicMaterial color="#3b82f6" wireframe transparent opacity={0.08} />
      </mesh>
    </group>
  );
}

export function ThreeBackground() {
  return (
    <div style={{ position: 'absolute', top: 0, left: 0, width: '100vw', height: '100%', zIndex: 0, pointerEvents: 'none' }}>
      <Canvas 
        camera={{ position: [0, 0, 5], fov: 45 }} 
        gl={{ alpha: true, antialias: false, powerPreference: "high-performance" }}
        dpr={[1, 1.5]}
      >
        <AbstractWireframe />
      </Canvas>
    </div>
  );
}
