.PHONY: package

PACKAGE_FILE_NAME = puppet.tar.gz
BUILD_DIR = $(CURDIR)/dist/
PACKAGE_FILE_PATH = $(addprefix $(BUILD_DIR), $(PACKAGE_FILE_NAME))

modules: Puppetfile
	bolt puppetfile install

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

package: $(PACKAGE_FILE_PATH)
$(PACKAGE_FILE_PATH): modules $(BUILD_DIR)
	tar -zcvf $(PACKAGE_FILE_PATH) --exclude='$(BUILD_DIR:$(CURDIR)/%=%)' .
