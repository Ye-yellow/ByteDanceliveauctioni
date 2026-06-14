import gsap from 'gsap';
import { useGSAP } from '@gsap/react';

// Keep React-specific GSAP setup in one place so feature components can stay focused.
gsap.registerPlugin(useGSAP);

export { gsap, useGSAP };
