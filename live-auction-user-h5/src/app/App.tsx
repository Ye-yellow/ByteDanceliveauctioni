import { Router } from './router';
import { AuthSessionProvider } from '../shared/auth/AuthSessionProvider';
import { useDragScrollAmounts } from '../shared/hooks/useDragScrollAmounts';
import './styles.css';
import './recovered-overlays.css';
import './live-product-carousel.css';

export default function App() {
  useDragScrollAmounts();
  return <AuthSessionProvider><Router /></AuthSessionProvider>;
}
