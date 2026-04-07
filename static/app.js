async function init() {
  const msgEl = document.getElementById('msg')
  try {
    const savedToken = sessionStorage.getItem('admin_token') || ''
    if (savedToken) document.getElementById('admin_token').value = savedToken

    const res = await fetch('/admin/config', { headers: savedToken ? { 'Authorization': 'Bearer ' + savedToken } : {} })
    if (!res.ok) {
      msgEl.innerText = '无法加载配置';
      return
    }
    const cfg = await res.json()

    document.getElementById('interface').value = cfg.Interface || ''
    const cf = cfg.Cloudflare || {}
    document.getElementById('cf_token').value = cf.Token || ''
    document.getElementById('cf_zone_id').value = cf.ZoneID || ''
    document.getElementById('cf_record_id').value = cf.RecordID || ''
    document.getElementById('cf_name').value = cf.Name || ''
    document.getElementById('cf_ttl').value = cf.TTL || ''

    const rt = cfg.Runtime || {}
    document.getElementById('update_interval').value = rt.UpdateInterval || ''
    document.getElementById('admin_addr').value = rt.AdminAddr || ''

    const nt = cfg.Notify || {}
    document.getElementById('wecom_webhook').value = nt.WecomWebhook || ''

    document.getElementById('cfgForm').addEventListener('submit', async (e) => {
      e.preventDefault()
      const enteredToken = document.getElementById('admin_token').value || ''
      if (enteredToken) sessionStorage.setItem('admin_token', enteredToken)
      const newCfg = {
        Interface: document.getElementById('interface').value,
        Cloudflare: {
          Token: document.getElementById('cf_token').value,
          ZoneID: document.getElementById('cf_zone_id').value,
          RecordID: document.getElementById('cf_record_id').value,
          Name: document.getElementById('cf_name').value,
          TTL: Number(document.getElementById('cf_ttl').value) || 1
        },
        Runtime: {
          UpdateInterval: Number(document.getElementById('update_interval').value) || 60,
          AdminAddr: document.getElementById('admin_addr').value || ':8080'
        },
        Notify: {
          WecomWebhook: document.getElementById('wecom_webhook').value
        }
      }

      const headers = { 'Content-Type': 'application/json' }
      const authToken = document.getElementById('admin_token').value || ''
      if (authToken) headers['Authorization'] = 'Bearer ' + authToken

      const resp = await fetch('/admin/config', {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(newCfg)
      })
      if (resp.ok) {
        msgEl.innerText = '已保存'
      } else {
        const txt = await resp.text()
        msgEl.innerText = '保存失败: ' + txt
      }
    })
    // backups
    async function loadBackups() {
      const token = sessionStorage.getItem('admin_token') || ''
      const headers = token ? { 'Authorization': 'Bearer ' + token } : {}
      const res = await fetch('/admin/backups', { headers })
      if (!res.ok) {
        document.getElementById('backupsList').innerHTML = '<li>无法获取备份</li>'
        return
      }
      const list = await res.json()
      const ul = document.getElementById('backupsList')
      ul.innerHTML = ''
      list.forEach((n) => {
        const li = document.createElement('li')
        const btn = document.createElement('button')
        btn.innerText = '恢复'
        btn.addEventListener('click', async () => {
          if (!confirm('确认恢复备份: ' + n + ' ?')) return
          const r = await fetch('/admin/backups/restore', {
            method: 'POST',
            headers: Object.assign({ 'Content-Type': 'application/json' }, headers),
            body: JSON.stringify({ file: n })
          })
          if (r.ok) {
            alert('已恢复: ' + n)
            window.location.reload()
          } else {
            const txt = await r.text()
            alert('恢复失败: ' + txt)
          }
        })
        li.innerText = n + ' '
        li.appendChild(btn)
        ul.appendChild(li)
      })
    }

    document.getElementById('refreshBackups').addEventListener('click', loadBackups)
    loadBackups()
  } catch (err) {
    msgEl.innerText = '错误: ' + err.message
  }
}

window.addEventListener('load', init)
