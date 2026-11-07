# Include from the top-level Makefile: -include tests/vm/vm.mk
VM := python3 tests/vm/vm.py

.PHONY: vm-doctor vm-up vm-down vm-destroy vm-recreate vm-status vm-ssh vm-sync vm-build vm-test vm-test-clean vm-gui vm-snapshot vm-restore vm-restart

vm-doctor:      ; @$(VM) doctor
vm-up:          ; @$(VM) up
vm-down:        ; @$(VM) stop
vm-destroy:     ; @$(VM) destroy
vm-recreate:    ; @$(VM) recreate
vm-status:      ; @$(VM) status
vm-ssh:         ; @$(VM) ssh
vm-sync:        ; @$(VM) sync
vm-build:       ; @$(VM) sync && $(VM) build
vm-test:        ; @$(VM) test
vm-test-clean:  ; @VM_RESTORE=1 $(VM) test
vm-gui:         ; @$(VM) gui on
vm-snapshot:    ; @$(VM) snapshot
vm-restore:     ; @$(VM) restore
vm-restart:     ; @$(VM) sync && $(VM) build && $(VM) gui on