import React, { useEffect, useMemo, useRef, useState } from 'react';
import { FaChevronDown, FaSignOutAlt, FaUserCircle } from 'react-icons/fa';
import ThemeToggle from './ThemeToggle';

const AvatarMenu = ({ authSession, themePreference, onThemeChange }) => {
  const [open, setOpen] = useState(false);
  const menuRef = useRef(null);
  const triggerRef = useRef(null);

  useEffect(() => {
    const handlePointerDown = (event) => {
      if (!menuRef.current || !triggerRef.current) {
        return;
      }

      if (
        !menuRef.current.contains(event.target) &&
        !triggerRef.current.contains(event.target)
      ) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  const displayName = useMemo(() => {
    if (authSession?.username) return authSession.username;
    if (authSession?.email) return authSession.email;
    return authSession?.mode === 'proxy' ? 'Guest' : 'Local';
  }, [authSession]);

  const avatarLabel = useMemo(() => {
    const source = displayName.trim();
    if (!source) return '?';
    return source.charAt(0).toUpperCase();
  }, [displayName]);

  const showLogout = authSession?.mode === 'proxy' && authSession?.authenticated && authSession?.logout_url;

  return (
    <div className="avatar-menu no-print">
      <button
        ref={triggerRef}
        type="button"
        className={`avatar-menu-trigger${open ? ' is-open' : ''}`}
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <span className="avatar-menu-badge" aria-hidden="true">
          {authSession?.authenticated ? avatarLabel : <FaUserCircle />}
        </span>
        <span className="avatar-menu-meta">
          <span className="avatar-menu-name">{displayName}</span>
          <span className="avatar-menu-role">{authSession?.role || 'viewer'}</span>
        </span>
        <FaChevronDown className={`avatar-menu-chevron${open ? ' is-open' : ''}`} aria-hidden="true" />
      </button>

      {open && (
        <div ref={menuRef} className="avatar-menu-panel" role="menu">
          <div className="avatar-menu-section">
            <div className="avatar-menu-section-label">Appearance</div>
            <ThemeToggle value={themePreference} onChange={onThemeChange} />
          </div>

          {showLogout && (
            <div className="avatar-menu-section">
              <a className="avatar-menu-action" href={authSession.logout_url}>
                <FaSignOutAlt aria-hidden="true" />
                <span>Log out</span>
              </a>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default AvatarMenu;
