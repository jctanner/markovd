import { useState, useEffect } from 'react';
import { api } from '../api';
import type { VolumeDefault, PVCInfo, SecretInfo } from '../api';

export default function Settings() {
  const [volumes, setVolumes] = useState<VolumeDefault[]>([]);
  const [secrets, setSecrets] = useState<VolumeDefault[]>([]);
  const [availablePVCs, setAvailablePVCs] = useState<PVCInfo[]>([]);
  const [availableSecrets, setAvailableSecrets] = useState<SecretInfo[]>([]);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    api.getPreferences().then((prefs) => {
      setVolumes(prefs.default_volumes);
      setSecrets(prefs.default_secrets);
    }).catch(() => {});
    api.listPVCs().then(setAvailablePVCs).catch(() => {});
    api.listSecrets().then(setAvailableSecrets).catch(() => {});
  }, []);

  const addVolume = () => setVolumes([...volumes, { name: '', mount_path: '' }]);
  const removeVolume = (i: number) => setVolumes(volumes.filter((_, idx) => idx !== i));
  const updateVolume = (i: number, field: keyof VolumeDefault, val: string) => {
    const updated = [...volumes];
    if (field === 'name' && !updated[i].mount_path) {
      updated[i] = { ...updated[i], name: val, mount_path: `/mnt/${val}` };
    } else {
      updated[i] = { ...updated[i], [field]: val };
    }
    setVolumes(updated);
  };

  const addSecret = () => setSecrets([...secrets, { name: '', mount_path: '' }]);
  const removeSecret = (i: number) => setSecrets(secrets.filter((_, idx) => idx !== i));
  const updateSecret = (i: number, field: keyof VolumeDefault, val: string) => {
    const updated = [...secrets];
    if (field === 'name' && !updated[i].mount_path) {
      updated[i] = { ...updated[i], name: val, mount_path: `/etc/secrets/${val}` };
    } else {
      updated[i] = { ...updated[i], [field]: val };
    }
    setSecrets(updated);
  };

  const selectedPVCNames = new Set(volumes.map(v => v.name));
  const selectedSecretNames = new Set(secrets.map(s => s.name));

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const cleanVolumes = volumes.filter(v => v.name && v.mount_path);
      const cleanSecrets = secrets.filter(s => s.name && s.mount_path);
      await api.updatePreferences({
        default_volumes: cleanVolumes,
        default_secrets: cleanSecrets,
      });
      setVolumes(cleanVolumes);
      setSecrets(cleanSecrets);
      setMessage({ type: 'success', text: 'Preferences saved' });
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to save' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Settings</h1>
      </div>

      <div className="settings-form">
        {message && (
          <div className={message.type === 'success' ? 'msg-success' : 'msg-error'}>
            {message.text}
          </div>
        )}

        <div className="card" style={{ marginBottom: 24 }}>
          <div className="card-title">Default PVC Volumes</div>
          <p className="settings-desc">
            These PVC volumes will be pre-selected on the Trigger Run page.
          </p>
          {volumes.map((v, i) => (
            <div key={i} className="defaults-row">
              <select
                className="form-select"
                value={v.name}
                onChange={(e) => updateVolume(i, 'name', e.target.value)}
              >
                <option value="">Select PVC...</option>
                {availablePVCs.map((p) => (
                  <option
                    key={p.name}
                    value={p.name}
                    disabled={p.name !== v.name && selectedPVCNames.has(p.name)}
                  >
                    {p.name} ({p.status})
                  </option>
                ))}
              </select>
              <input
                type="text"
                className="form-input"
                placeholder="/mnt/..."
                value={v.mount_path}
                onChange={(e) => updateVolume(i, 'mount_path', e.target.value)}
              />
              <button type="button" className="btn btn-danger btn-sm" onClick={() => removeVolume(i)}>
                x
              </button>
            </div>
          ))}
          <button type="button" className="btn btn-ghost btn-sm" onClick={addVolume}>
            + Add PVC
          </button>
        </div>

        <div className="card" style={{ marginBottom: 24 }}>
          <div className="card-title">Default Secret Volumes</div>
          <p className="settings-desc">
            These secret volumes will be pre-selected on the Trigger Run page.
          </p>
          {secrets.map((s, i) => (
            <div key={i} className="defaults-row">
              <select
                className="form-select"
                value={s.name}
                onChange={(e) => updateSecret(i, 'name', e.target.value)}
              >
                <option value="">Select secret...</option>
                {availableSecrets.map((sec) => (
                  <option
                    key={sec.name}
                    value={sec.name}
                    disabled={sec.name !== s.name && selectedSecretNames.has(sec.name)}
                  >
                    {sec.name} ({sec.type})
                  </option>
                ))}
              </select>
              <input
                type="text"
                className="form-input"
                placeholder="/etc/secrets/..."
                value={s.mount_path}
                onChange={(e) => updateSecret(i, 'mount_path', e.target.value)}
              />
              <button type="button" className="btn btn-danger btn-sm" onClick={() => removeSecret(i)}>
                x
              </button>
            </div>
          ))}
          <button type="button" className="btn btn-ghost btn-sm" onClick={addSecret}>
            + Add Secret
          </button>
        </div>

        <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving...' : 'Save Preferences'}
        </button>
      </div>
    </div>
  );
}
