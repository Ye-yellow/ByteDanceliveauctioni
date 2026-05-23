import { Router } from './router';
import { AuthSessionProvider } from '../shared/auth/AuthSessionProvider';
import './styles.css';
export default function App() { return <AuthSessionProvider><Router /></AuthSessionProvider>; }
