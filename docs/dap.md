Soulsolid provides synchronization capabilities for Digital Audio Players (DAPs) using a high-performance Go implementation of rsync. This ensures efficient, reliable file transfers between your music library and portable devices. 
Note that you device should be connected to the same machine running Musiscarr or Soulsolid shuold be able to access the mounted device one way or another. You can do `NFS` or `sshfs`, get creative.

### Device Configuration

Each DAP is identified by its UUID, which you can find using the `lsblk -f` command once the device is connected:

```bash
> lsblk -f

 NAME        FSTYPE FSVER LABEL UUID                                 FSAVAIL FSUSE% MOUNTPOINTS
 sda
 └─sda1      vfat   FAT32 IPOD  8722-177E                             631.3G    34% /mnt/ipod
```

The UUID (like ` 8722-177E` or `cced04fa-857b-4c49-afb8-d530c8e802e2`) uniquely identifies your device.

Soulsolid automatically detects if the device is mounted and enables synchronization for it.

### Sync Path Configuration

The `sync_path` option specifies the folder that will be created and synchronized on your DAP:

- **Location**: The path is relative to the device's mount point root
- **Automatic Creation**: Soulsolid will create the folder if it doesn't exist
- **Example**: If your device is mounted at `/mnt/ipod` and `sync_path` is `music`, files will be copied to `/mnt/ipod/music`

### Example Configuration

```yaml
sync:
  enabled: true
  devices:
    - uuid: 8722-177E
      name: iPod    
      sync_path: Soulsolid
```

This configuration will sync your music library to:

- `/mnt/ipod/Soulsolid` (when iPod is mounted at `/mnt/ipod`)


### Logs

Device sync uses the job feature therefore logs can by obtained in in the `/jobs/<job-id>/logs` endpoint or in the `/ui/jobs` section

